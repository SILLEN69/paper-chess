"""Original monochrome SVG chess pieces, with no remote dependencies."""
from pathlib import Path
assets=Path(__file__).parent/'assets'
assets.mkdir(exist_ok=True)
shapes={
 'p':'<circle cx="40" cy="21" r="10"/><path d="M31 32h18l-2 8c0 8 5 13 9 20H24c4-7 9-12 9-20z"/><path d="M24 61h32l5 9H19z"/>',
 'r':'<path d="M22 12h9v9h7v-9h8v9h7v-9h9v20l-9 6 3 23H24l3-23-5-6z"/><path d="M24 61h32l5 9H19z"/><path class="detail" d="M26 32h28M28 42h24"/>',
 'b':'<path d="M40 10c-6 7-15 13-15 24 0 8 5 13 11 15l-7 12h22l-7-12c6-2 11-7 11-15 0-11-9-17-15-24z"/><path class="detail" d="M43 22l-9 13"/><path d="M24 61h32l5 9H19z"/><circle cx="40" cy="10" r="3"/>',
 'n':'<path d="M25 60c1-12 5-23 16-30l-9 3-8 9-10-4 8-18 12-8 12 2 6-7 5 17c9 10 10 23 4 36z"/><path class="detail" d="M42 23l-7 1M24 36l7-3M45 34c-9 6-12 13-12 21"/><path d="M24 61h32l5 9H19z"/>',
 'q':'<path d="M20 24l10 10 10-16 10 16 10-10-7 27H27z"/><path d="M28 51h24l4 10H24z"/><path d="M24 61h32l5 9H19z"/><circle cx="19" cy="22" r="4"/><circle cx="40" cy="15" r="4"/><circle cx="61" cy="22" r="4"/><path class="detail" d="M30 45h20"/>',
 'k':'<path d="M37 8h6v7h7v6h-7v9h-6v-9h-7v-6h7z"/><path d="M25 33c0-8 9-10 15-3 6-7 15-5 15 3 0 8-7 12-8 19H33c-1-7-8-11-8-19z"/><path d="M30 52h20l6 9H24z"/><path d="M24 61h32l5 9H19z"/><path class="detail" d="M32 44h16"/>',
}
for color in ['w','b']:
 for role,shape in shapes.items():
  fill,detail=('white','#171717') if color=='w' else ('#171717','white')
  svg=f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><style>.detail{{fill:none;stroke:{detail};stroke-width:2.4}}</style><g fill="{fill}" stroke="#171717" stroke-width="2.5" stroke-linejoin="round" stroke-linecap="round">{shape}</g></svg>'
  (assets/f'{color}{role}.svg').write_text(svg,encoding='utf-8')
