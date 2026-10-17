pragma ComponentBehavior: Bound
import QtQuick
Rectangle {
 id:root; anchors.fill:parent; z:100000; color:"white";focus:true
 rotation:state.rotated?180:0
 property real u:Math.min(width/1404,height/1872)
 property real boardSize:Math.min(width-120*u,height-600*u)
 property var state:({board:[],moves:[],legal:[],last:[],hint:[],guide:true,level:1,human:"w",mode:"computer",lesson:-1,rotated:false})
 property string modal:""
 property string newMode:"computer"
 property int newLevel:1
 property string newHuman:"w"
 property string connectionError:""
 property bool requesting:false
 property var queuedRequests:[]
 readonly property var levelNames:["","Beginner","Practice","Challenge"]
 readonly property real boardLeft:(width-boardSize)/2
 readonly property real boardTop:270*u
 function contains(list,item){return list&&list.indexOf(item)>=0}
 function request(path,data){
  if(requesting){if(data&&queuedRequests.length<12)queuedRequests.push({path:path,data:data});return}
  requesting=true
  var req=new XMLHttpRequest()
  req.onreadystatechange=function(){if(req.readyState===XMLHttpRequest.DONE){requesting=false;if(req.status===200){try{root.state=JSON.parse(req.responseText);connectionError=""}catch(e){connectionError="Could not read the board. Tap Retry."}}else connectionError="The game is reconnecting. Tap Retry if this stays visible.";if(queuedRequests.length){var next=queuedRequests.shift();request(next.path,next.data)}}}
  req.open(data?"POST":"GET","http://127.0.0.1:9148"+path)
  if(data)req.setRequestHeader("Content-Type","application/json")
  req.send(data?JSON.stringify(data):"")
 }
 function act(action,extra){var data=extra||{};data.action=action;request("/api/action",data)}
 function openNew(){newMode=state.mode==="local"?"local":"computer";newLevel=state.level||1;newHuman=state.human||"w";modal="new"}
 function quit(){var req=new XMLHttpRequest();req.open("POST","http://127.0.0.1:9148/quit");req.send()}
 Component.onCompleted:request("/api/state")
 Timer{interval:450;repeat:true;running:root.state.busy===true;onTriggered:root.request("/api/state")}
 component ActionButton:Rectangle{
  id:button;property string label:"";property bool selected:false;property bool subdued:false;signal clicked()
  color:selected?"#1b1b1b":"white";border.color:subdued?"#aaaaaa":"#1b1b1b";border.width:2*root.u;radius:4*root.u;implicitHeight:78*root.u
  Text{anchors.centerIn:parent;text:button.label;color:button.selected?"white":"#202020";font.pixelSize:30*root.u;font.weight:Font.Medium}
  TapHandler{gesturePolicy:TapHandler.ReleaseWithinBounds;onTapped:button.clicked()}
 }
 ActionButton{x:60*root.u;y:40*root.u;width:190*root.u;height:72*root.u;label:"‹  Notes";subdued:true;onClicked:root.quit()}
 Text{anchors.horizontalCenter:parent.horizontalCenter;y:43*root.u;text:"Paper Chess";font.family:"Noto Serif";font.pixelSize:54*root.u;font.weight:Font.Medium;color:"#151515"}
 ActionButton{x:root.width-width-60*root.u;y:40*root.u;width:230*root.u;height:72*root.u;label:root.state.lesson>=0?"Back to game":"New game";subdued:true;onClicked:{if(root.state.lesson>=0)root.act("resume");else root.openNew()}}
 Rectangle{x:60*root.u;y:140*root.u;width:parent.width-120*root.u;height:2*root.u;color:"#202020"}
 Text{x:root.boardLeft;y:176*root.u;text:root.state.lesson>=0?"LEARN  /  "+(root.state.lesson+1)+" OF 9":root.state.mode==="local"?"LOCAL  /  TWO PLAYERS":"COMPUTER  /  "+root.levelNames[root.state.level].toUpperCase();font.pixelSize:25*root.u;font.letterSpacing:2*root.u;color:"#5a5a5a"}
 Text{x:root.boardLeft;y:214*root.u;text:root.state.lessonData?root.state.lessonData.title:root.state.mode==="local"?"Take your time. Play together.":"You play "+(root.state.human==="b"?"Black":"White")+".";font.pixelSize:30*root.u;color:"#3a3a3a"}
 Text{anchors.right:board.right;y:174*root.u;text:root.state.status||"Getting your board ready…";font.pixelSize:33*root.u;font.weight:Font.DemiBold;color:"#171717"}
 Text{anchors.right:board.right;y:216*root.u;text:root.state.moves.length?"Last move: "+root.state.moves[root.state.moves.length-1]:"Tap a piece, then a destination.";font.pixelSize:26*root.u;color:"#5a5a5a"}
 Item{
  id:board;x:root.boardLeft;y:root.boardTop;width:root.boardSize;height:width
  Repeater{model:64
   Rectangle{
    required property int index
    property int sourceIndex:root.state.flipped?63-index:index
    property var cell:root.state.board[sourceIndex]||({square:"",piece:"",color:""})
    property bool legal:root.contains(root.state.legal,cell.square)
    x:(index%8)*board.width/8;y:Math.floor(index/8)*board.height/8;width:board.width/8;height:width
    color:(Math.floor(index/8)+index%8)%2?"#c4c4c4":"#fafaf8"
    Rectangle{anchors.fill:parent;anchors.margins:5*root.u;color:"transparent";visible:root.contains(root.state.last,parent.cell.square);border.width:3*root.u;border.color:"#777777"}
    Image{anchors.centerIn:parent;width:parent.width*.87;height:width;source:parent.cell.piece?"file:///home/root/paper-chess/assets/"+parent.cell.color+parent.cell.piece+".svg":"";fillMode:Image.PreserveAspectFit;smooth:true}
    Rectangle{anchors.fill:parent;anchors.margins:4*root.u;color:"transparent";visible:root.state.selected===parent.cell.square;border.width:7*root.u;border.color:"#151515"}
    Rectangle{anchors.centerIn:parent;width:parent.cell.piece?parent.width*.78:24*root.u;height:width;radius:width/2;visible:parent.legal&&root.state.guide;color:parent.cell.piece?"transparent":"#565656";border.color:"#383838";border.width:parent.cell.piece?5*root.u:0}
    Rectangle{anchors.fill:parent;anchors.margins:12*root.u;color:"transparent";visible:root.contains(root.state.hint,parent.cell.square);border.width:5*root.u;border.color:"#171717"}
    Text{x:12*root.u;y:7*root.u;visible:root.state.check===parent.cell.square;text:"CHECK";font.pixelSize:19*root.u;font.bold:true;color:"#111111";style:Text.Outline;styleColor:"white"}
    TapHandler{gesturePolicy:TapHandler.ReleaseWithinBounds;onTapped:root.act("tap",{square:parent.cell.square})}
   }
  }
  Rectangle{anchors.fill:parent;color:"transparent";border.width:2*root.u;border.color:"#1a1a1a"}
  Repeater{model:8;Text{required property int index;x:-33*root.u;y:(index+.5)*board.width/8-height/2;text:root.state.flipped?(index+1):(8-index);font.pixelSize:24*root.u;color:"#555555"}}
  Repeater{model:8;Text{required property int index;x:(index+.5)*board.width/8-width/2;y:board.height+7*root.u;text:"abcdefgh"[root.state.flipped?7-index:index];font.pixelSize:24*root.u;color:"#555555"}}
 }
 Text{id:adviceHeading;x:root.boardLeft;y:board.y+board.height+57*root.u;text:root.connectionError?"CONNECTION":root.state.lesson>=0?(root.state.lessonDone?"WELL PLAYED":"YOUR NEXT STEP"):root.state.hint.length?"A MOVE TO CONSIDER":"AT THE BOARD";font.pixelSize:24*root.u;font.letterSpacing:2*root.u;font.weight:Font.DemiBold;color:"#333333"}
 Text{x:root.boardLeft;y:adviceHeading.y+37*root.u;width:board.width;height:139*root.u;text:root.connectionError||(root.state.lessonData&&!root.state.lessonDone?root.state.lessonData.body:root.state.message)||"Control the centre, develop your knights and bishops, and keep your king safe. Tap Learn for a guided first game.";font.pixelSize:30*root.u;lineHeight:1.13;wrapMode:Text.WordWrap;color:"#292929"}
 Row{x:root.boardLeft;y:root.height-103*root.u;spacing:18*root.u
  ActionButton{width:(board.width-54*root.u)/4;label:root.connectionError?"Retry":root.state.lesson>=0?"Try again":"Undo";onClicked:{if(root.connectionError)root.request("/api/state");else root.act("undo")}}
  ActionButton{width:(board.width-54*root.u)/4;label:root.state.busy?"Thinking…":"Hint";onClicked:root.act("hint")}
  ActionButton{width:(board.width-54*root.u)/4;label:root.state.lesson>=0?"Lessons":"Learn";onClicked:root.modal="lessons"}
  ActionButton{width:(board.width-54*root.u)/4;label:root.state.lesson>=0?(root.state.lesson<8?"Next lesson":"Play a game"):"Flip screen";onClicked:{if(root.state.lesson>=0){if(root.state.lesson<8)root.act("lesson",{index:root.state.lesson+1});else root.act("resume")}else root.act("flip")}}
 }
 Rectangle{
  anchors.fill:parent;z:1000;visible:root.modal!==""||root.state.promotion===true;color:Qt.rgba(1,1,1,.91)
  TapHandler{gesturePolicy:TapHandler.ReleaseWithinBounds}
  Rectangle{
   id:sheet;width:Math.min(root.width-120*root.u,1130*root.u);height:root.modal==="lessons"?1120*root.u:root.state.promotion?480*root.u:960*root.u;anchors.centerIn:parent;color:"white";border.color:"#1d1d1d";border.width:3*root.u
   Text{x:55*root.u;y:48*root.u;text:root.state.promotion?"Choose your new piece":root.modal==="lessons"?"Learn by playing":"A new game";font.family:"Noto Serif";font.pixelSize:49*root.u;color:"#181818"}
   ActionButton{anchors.right:parent.right;anchors.rightMargin:40*root.u;y:45*root.u;width:160*root.u;height:64*root.u;label:"Close";visible:!root.state.promotion;subdued:true;onClicked:root.modal=""}
   Item{
    anchors.fill:parent;visible:root.modal==="new"&&!root.state.promotion
    Text{x:55*root.u;y:137*root.u;width:parent.width-110*root.u;text:root.state.moves.length?"Your current game will be replaced when you press Start.":"An offline game, at your own pace.";font.pixelSize:29*root.u;wrapMode:Text.WordWrap;color:"#555555"}
    Row{x:55*root.u;y:220*root.u;spacing:20*root.u
     ActionButton{width:490*root.u;label:"Against the computer";selected:root.newMode==="computer";onClicked:root.newMode="computer"}
     ActionButton{width:490*root.u;label:"Two players";selected:root.newMode==="local";onClicked:root.newMode="local"}
    }
    Text{x:55*root.u;y:355*root.u;text:root.newMode==="computer"?"CHOOSE YOUR PACE":"PLAY ACROSS THE TABLE";font.pixelSize:25*root.u;font.letterSpacing:2*root.u;color:"#555555"}
    Row{x:55*root.u;y:405*root.u;spacing:20*root.u;visible:root.newMode==="computer";Repeater{model:3;ActionButton{required property int index;width:320*root.u;label:root.levelNames[index+1];selected:root.newLevel===index+1;onClicked:root.newLevel=index+1}}}
    Text{x:55*root.u;y:419*root.u;width:parent.width-110*root.u;visible:root.newMode==="local";text:"Share the board. Use Flip screen to turn the whole interface toward the other player.";font.pixelSize:31*root.u;wrapMode:Text.WordWrap;color:"#333333"}
    Text{x:55*root.u;y:555*root.u;text:root.newMode==="computer"?"YOUR PIECES":"READY TO LEARN?";font.pixelSize:25*root.u;font.letterSpacing:2*root.u;color:"#555555"}
    Row{x:55*root.u;y:607*root.u;spacing:20*root.u;visible:root.newMode==="computer"
     ActionButton{width:490*root.u;label:"White · you move first";selected:root.newHuman==="w";onClicked:root.newHuman="w"}
     ActionButton{width:490*root.u;label:"Black · computer first";selected:root.newHuman==="b";onClicked:root.newHuman="b"}
    }
    Text{x:55*root.u;y:607*root.u;width:parent.width-110*root.u;visible:root.newMode==="local";text:"Legal moves, hints and undo are available to both players. Learn has nine short lessons.";font.pixelSize:31*root.u;wrapMode:Text.WordWrap;color:"#333333"}
    ActionButton{x:55*root.u;y:804*root.u;width:parent.width-110*root.u;height:94*root.u;label:"Start game";selected:true;onClicked:{root.modal="";root.act("new",{mode:root.newMode,level:root.newLevel,human:root.newHuman})}}
   }
   Item{
    anchors.fill:parent;visible:root.modal==="lessons"&&!root.state.promotion
    Text{x:55*root.u;y:129*root.u;text:"Nine small steps. Your saved game stays safe.";font.pixelSize:29*root.u;color:"#555555"}
    Column{x:55*root.u;y:210*root.u;width:parent.width-110*root.u
     Repeater{
      model:["The rook","The bishop","The knight","The pawn","Keep your king safe","Your first checkmate","Pawn promotion","En passant","Look for free material"]
      Item{
       required property int index;required property string modelData;width:parent.width;height:92*root.u
       Text{x:0;y:21*root.u;text:(parent.index+1<10?"0":"")+(parent.index+1);font.pixelSize:27*root.u;color:"#777777"}
       Text{x:78*root.u;y:18*root.u;text:parent.modelData;font.pixelSize:33*root.u;color:"#171717"}
       Text{anchors.right:parent.right;y:23*root.u;text:root.state.completed>parent.index?"Complete  ›":"›";font.pixelSize:27*root.u;color:"#555555"}
       Rectangle{anchors.bottom:parent.bottom;width:parent.width;height:1*root.u;color:"#cccccc"}
       TapHandler{gesturePolicy:TapHandler.ReleaseWithinBounds;onTapped:{root.modal="";root.act("lesson",{index:parent.index})}}
      }
     }
    }
   }
   Row{
    x:55*root.u;y:190*root.u;spacing:20*root.u;visible:root.state.promotion===true
    Repeater{model:["q","r","b","n"]
     ActionButton{
      required property string modelData;width:240*root.u;height:210*root.u;label:""
      Image{anchors.horizontalCenter:parent.horizontalCenter;y:20*root.u;width:130*root.u;height:width;source:"file:///home/root/paper-chess/assets/"+(root.state.turn==="White"?"w":"b")+parent.modelData+".svg"}
      Text{anchors.horizontalCenter:parent.horizontalCenter;y:158*root.u;text:({q:"Queen",r:"Rook",b:"Bishop",n:"Knight"})[parent.modelData];font.pixelSize:28*root.u;color:"#222222"}
      onClicked:root.act("promote",{piece:modelData})
     }
    }
   }
  }
 }
}
