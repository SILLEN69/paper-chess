//go:build ignore

/* MIT. Qt ABI bridge adapted from Inkling, copyright 2026 Nathan Marlor.
 * Runs only in an explicitly launched chess session. No hooks or note mutations.
 */
#define _GNU_SOURCE
#include <dlfcn.h>
#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <stdatomic.h>

typedef void (*fn_list)(void *);
typedef void (*fn_children)(void *,const void *);
typedef const char *(*fn_class)(const void *);
typedef void *(*fn_qml)(const void *);
typedef void (*fn_ctor)(void *,void *,void *);
typedef void (*fn_data)(void *,const void *,const void *);
typedef void *(*fn_create)(void *,void *);
typedef void (*fn_ba)(void *,const char *,int);
typedef void (*fn_one)(void *);
typedef void (*fn_parent)(void *,void *);
typedef char (*fn_queue)(void *,void *,int,void *);
typedef void (*fn_enable)(void*,int);
static fn_list allwindows;
static fn_children childitems;
static fn_class classname;
static fn_qml engineof,contextof;
static fn_ctor ctor;
static fn_data setdata;
static fn_create create;
static fn_ba bytesctor;
static fn_one urlctor,urldtor;
static fn_parent parentitem,parentobj;
static fn_queue queue;
static fn_enable enableitem;
static int (*closingdown)(void);
static void **appself;
static void *rootitem;
static _Atomic int job_pending;
static _Atomic int ready;
static void *chesswindow;
static char component[128] __attribute__((aligned(16)));
static char bytearray[64] __attribute__((aligned(16)));
static int stopping(void) {
    return access("/run/paper-chess/stopping",F_OK)==0 || (closingdown&&closingdown());
}
static int quiesce_if_stopping(void) {
    if(!stopping())return 0;
    for(int i=0;i<100&&atomic_load(&job_pending);i++)usleep(20000);
    if(!atomic_load(&job_pending)){
        FILE *file=fopen("/run/paper-chess/ui-quiesced","w");
        if(file){fputs("quiesced\n",file);fclose(file);}
    }
    return 1;
}

static const char *cls(void *o) {
    if(!o) return "";
    void *(*meta)(const void*)=(void*(*)(const void*))(*(void***)o)[0];
    return classname(meta(o));
}
static void *find_engine_item(void *item,int depth) {
    if(!item||depth>12) return NULL;
    if(engineof(item)) return item;
    void *list[3]={0}; childitems(list,item);
    void **items=list[1]; long count=(long)list[2];
    if(count<0||count>1000) return NULL;
    for(long i=0;i<count;i++){void *v=find_engine_item(items[i],depth+1);if(v)return v;}
    return NULL;
}
static void install_ui(void) {
    if(ready||stopping()) return;
    void *windows[3]={0}; allwindows(windows);
    void **wins=windows[1]; long count=(long)windows[2];
    void *(*content)(void*)=dlsym(RTLD_DEFAULT,"_ZNK12QQuickWindow11contentItemEv");
    if(!content||count<1||count>16) return;
    for(long i=0;i<count;i++) {
        if(!strstr(cls(wins[i]),"MainWindow")) continue;
        rootitem=content(wins[i]);
        void *contextitem=find_engine_item(rootitem,0);
        if(!contextitem)continue;
        FILE *file=fopen("/home/root/paper-chess/Main.qml","rb");
        if(!file)return;
        fseek(file,0,SEEK_END);long size=ftell(file);rewind(file);
        if(size<1||size>256000){fclose(file);return;}
        char *text=calloc(1,size+1);if(!text){fclose(file);return;}
        size_t read=fread(text,1,size,file);fclose(file);
        if(read!=(size_t)size){free(text);return;}
        ctor(component,engineof(contextitem),NULL);
        bytesctor(bytearray,text,size);free(text);
        char url[64]={0};urlctor(url);setdata(component,bytearray,url);urldtor(url);
        void *ui=create(component,contextof(contextitem));
        if(!ui){fprintf(stderr,"[paper-chess] QML creation failed\n");ready=-1;return;}
        parentobj(ui,rootitem);parentitem(ui,rootitem);
        // A modal game must not pass pen/finger input through to a notebook.
        {void *children[3]={0};childitems(children,rootitem);void **items=children[1];long n=(long)children[2];if(n>=0&&n<1000)for(long k=0;k<n;k++)if(items[k]!=ui)enableitem(items[k],0);}
        chesswindow=wins[i];ready=1;
        FILE *marker=fopen("/run/paper-chess/ui-ready","w");if(marker){fputs("ready\n",marker);fclose(marker);}
        fprintf(stderr,"[paper-chess] UI attached to %s\n",cls(wins[i]));
        return;
    }
}
static void exec_slot(void *a1,void *a2,void **a3,int which,char *ret) {
    (void)a2;(void)a3;(void)ret;
    int operation=((unsigned long)a1<8)?(int)(unsigned long)a1:which;
    if(operation==1){install_ui();atomic_store(&job_pending,0);}
}
static void *slot[4]={(void*)exec_slot,(void*)exec_slot,0,0};
static void capture(void) {
    if(stopping())return;
    void (*grab)(void*,void*)=dlsym(RTLD_DEFAULT,"_ZN12QQuickWindow10grabWindowEv");
    const unsigned char *(*bits)(const void*)=dlsym(RTLD_DEFAULT,"_ZNK6QImage9constBitsEv");
    int (*width)(const void*)=dlsym(RTLD_DEFAULT,"_ZNK6QImage5widthEv");
    int (*height)(const void*)=dlsym(RTLD_DEFAULT,"_ZNK6QImage6heightEv");
    int (*stride)(const void*)=dlsym(RTLD_DEFAULT,"_ZNK6QImage12bytesPerLineEv");
    int (*format)(const void*)=dlsym(RTLD_DEFAULT,"_ZNK6QImage6formatEv");
    long (*size)(const void*)=dlsym(RTLD_DEFAULT,"_ZNK6QImage11sizeInBytesEv");
    void (*destroy)(void*)=dlsym(RTLD_DEFAULT,"_ZN6QImageD1Ev");
    if(!chesswindow||!grab||!bits||!width||!height||!stride||!format||!size||!destroy)return;
    char image[128] __attribute__((aligned(16)))={0};grab(image,chesswindow);
    int header[5]={width(image),height(image),stride(image),format(image),(int)size(image)};
    if(header[0]>0&&header[1]>0&&header[4]>0&&header[4]<20000000&&bits(image)){
        FILE *file=fopen("/home/root/paper-chess/frame.raw","wb");
        if(file){fwrite(header,4,5,file);fwrite(bits(image),1,header[4],file);fclose(file);}
    }
    destroy(image);
}
static void *worker(void *unused) {
    (void)unused;
    for(int i=0;i<30;i++){if(quiesce_if_stopping())return NULL;usleep(100000);}
    for(int i=0;i<20&&!ready;i++) {
        if(quiesce_if_stopping())return NULL;
        if(appself&&*appself&&!atomic_exchange(&job_pending,1)){
            if(!queue(*appself,slot,2,NULL))atomic_store(&job_pending,0);
        }
        for(int j=0;j<10;j++){if(quiesce_if_stopping())return NULL;usleep(100000);}
    }
    while(ready==1){
        if(quiesce_if_stopping())return NULL;
        if(access("/run/paper-chess/capture",F_OK)==0){capture();unlink("/run/paper-chess/capture");}
        sleep(1);
    }
    return NULL;
}
#define LOAD(variable,symbol) do{ *(void**)(&variable)=dlsym(RTLD_DEFAULT,symbol);if(!variable){fprintf(stderr,"[paper-chess] Missing %s\n",symbol);return;}}while(0)
__attribute__((constructor)) static void initialize(void) {
    char process[32]={0};FILE *comm=fopen("/proc/self/comm","r");
    if(!comm)return;
    fgets(process,sizeof process,comm);fclose(comm);
    if(strcmp(process,"xochitl\n"))return;
    LOAD(allwindows,"_ZN15QGuiApplication10allWindowsEv");
    LOAD(childitems,"_ZNK10QQuickItem10childItemsEv");
    LOAD(classname,"_ZNK11QMetaObject9classNameEv");
    LOAD(engineof,"_Z9qmlEnginePK7QObject");
    LOAD(contextof,"_Z10qmlContextPK7QObject");
    LOAD(ctor,"_ZN13QQmlComponentC1EP10QQmlEngineP7QObject");
    LOAD(setdata,"_ZN13QQmlComponent7setDataERK10QByteArrayRK4QUrl");
    LOAD(create,"_ZN13QQmlComponent6createEP11QQmlContext");
    LOAD(bytesctor,"_ZN10QByteArrayC1EPKci");
    LOAD(urlctor,"_ZN4QUrlC1Ev");LOAD(urldtor,"_ZN4QUrlD1Ev");
    LOAD(parentitem,"_ZN10QQuickItem13setParentItemEPS_");LOAD(parentobj,"_ZN7QObject9setParentEPS_");
    LOAD(enableitem,"_ZN10QQuickItem10setEnabledEb");
    LOAD(queue,"_ZN11QMetaObject16invokeMethodImplEP7QObjectPN9QtPrivate15QSlotObjectBaseEN2Qt14ConnectionTypeEPv");
    LOAD(appself,"_ZN16QCoreApplication4selfE");
    LOAD(closingdown,"_ZN16QCoreApplication11closingDownEv");
    pthread_t thread;pthread_create(&thread,NULL,worker,NULL);pthread_detach(thread);
    fprintf(stderr,"[paper-chess] Runtime bridge loaded\n");
}
