// [VXG RealForever]
// Native VexStream Music -> VexBoard Screen Lens exporter.
// Reads VexStream-owned semantic tokens and the lived HTML/CSS shell.
// Product JavaScript is never exported; data-vex-ref exists only in the inert copy.

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { PlaybackAct, PlaybackControl, PlaybackEffect, VexStreamPlaybackFamily } from "../web/features/playback/behavior.js";
import { QueueAct, QueueControl, QueueEffect, VexStreamQueueFamily } from "../web/features/queue/behavior.js";

export const VEXBOARD_SCREEN_LENS_SCHEMA="vexboard.screen-lens/v1";
export const VEXSTREAM_VEXBOARD_EXPORTER_VERSION="vexstream-vexboard-exporter/1";
const HERE=path.dirname(fileURLToPath(import.meta.url));
const V2=path.resolve(HERE,"..");
const SCREEN_REF="vexstream.desktop.library";
const FRAME_REF="vexstream.desktop.library.frame";
const SHELL_REF="vexstream.desktop.library.region.shell";
const PREVIOUS_ACTION="vexstream.player.previous";
const semanticIdentity=token=>token.familyRef+"."+token.ref;
const elementRef=identity=>"element."+identity;
const source=relative=>"v2:"+relative;

function viewHTML(sourceHTML){
  let html=String(sourceHTML);
  html=html.replace('<body data-vex-screen="vexstream.desktop.library" data-shell-current-view="home">','<body data-vex-screen="vexstream.desktop.library" data-shell-current-view="home" data-vex-structural-ref="'+FRAME_REF+'">');
  html=html.replace('<main class="shell">','<main class="shell" data-vex-structural-ref="'+SHELL_REF+'">');
  const rows=[
    ["data-vex-action",PREVIOUS_ACTION],
    ["data-vex-control",semanticIdentity(PlaybackControl.PlayPause)],
    ["data-vex-control",semanticIdentity(QueueControl.Next)],
    ["data-vex-control",semanticIdentity(QueueControl.Shuffle)],
    ["data-vex-control",semanticIdentity(QueueControl.Repeat)],
  ];
  for(const [attribute,identity] of rows){
    const before=attribute+'="'+identity+'"';
    const after=before+' data-vex-ref="'+elementRef(identity)+'"';
    html=html.replace(before,after);
  }
  return html;
}

function node(ref,face,role,label,options={}){
  return {ref,face,role,label,
    ...(options.symbol?{symbol:options.symbol}:{}),
    ...(options.summary?{summary:options.summary}:{}),
    ...(options.sourcePath?{source:source(options.sourcePath)}:{}),
  };
}
const edge=(from,to,label=null,kind="FLOW")=>({from,to,...(label?{label}:{}),kind});

function playbackPathway(){
  const p="pathway.vexstream.playback.play-pause";
  const ctl=p+".control", state=p+".state", act=p+".act", effect=p+".effect", projection=p+".projection", method=p+".player", connection=p+".connection", observation=p+".observation";
  return {
    pathwayRef:p,label:"Play / Pause",disposition:"SCAE_FAMILY",
    summary:"Playback owns Play/Pause meaning. Player composition may seed or load listening context before the same Playback Control enters the Family; provider success remains HTMLMediaElement truth.",
    returns:"Observed media truth -> current Playback State -> current Play/Pause presentation",returnRef:projection,
    nodes:[
      node(ctl,"BEHAVIOR","CONTROL","PlaybackControl.PlayPause",{symbol:semanticIdentity(PlaybackControl.PlayPause),sourcePath:"web/features/playback/behavior.js"}),
      node(state,"BEHAVIOR","STATE","PlaybackState",{symbol:VexStreamPlaybackFamily,summary:"no-media / media / ended semantic truth",sourcePath:"web/features/playback/behavior.js"}),
      node(act,"BEHAVIOR","ACT","Play / Pause",{symbol:PlaybackAct.Play.ref+" | "+PlaybackAct.Pause.ref,sourcePath:"web/features/playback/behavior.js"}),
      node(effect,"BEHAVIOR","EFFECT","Request play / pause / restart current",{symbol:PlaybackEffect.RequestPlay.ref+" | "+PlaybackEffect.RequestPause.ref+" | "+PlaybackEffect.RestartCurrent.ref,sourcePath:"web/features/playback/behavior.js"}),
      node(projection,"BEHAVIOR","PROJECTION","Play / Pause presentation",{sourcePath:"web/features/playback/presentation.js"}),
      node(method,"DEPTH","METHOD","Player.activatePlayPause()",{summary:"Higher composition seeds Queue or media only when listening context requires it.",sourcePath:"web/features/player/composition.js"}),
      node(connection,"DEPTH","CONNECTION","HTML media Playback connection",{symbol:"createHTMLMediaPlaybackConnection",summary:"Re-resolves Control and keeps request-pending distinct from provider acknowledgement.",sourcePath:"web/features/playback/connection.js"}),
      node(observation,"DEPTH","OBSERVATION","HTMLMediaElement truth",{summary:"paused, ended and play acknowledgement remain provider observations.",sourcePath:"web/features/playback/connection.js"}),
    ],
    edges:[
      edge(ctl,act,"resolves with current State"),edge(state,act,null,"DATA"),edge(act,effect,"Family resolves"),
      edge(effect,method,"realized through composition"),edge(method,connection),edge(connection,observation),
      edge(observation,state,"observed truth","RETURN"),edge(state,projection,"projects"),
    ],
  };
}

function queuePathway(key,control,actLabel,effectLabel,methodLabel){
  const p="pathway.vexstream.queue."+key;
  const ctl=p+".control", state=p+".state", act=p+".act", effect=p+".effect", projection=p+".projection", method=p+".player", realization=p+".queue-connection";
  return {
    pathwayRef:p,label:key[0].toUpperCase()+key.slice(1),disposition:"SCAE_FAMILY",
    summary:"Queue owns the Control/State/Act/Effect relationship; Player composition may carry current Queue position into Playback realization without becoming Queue meaning.",
    returns:"Current Queue State -> current control presentation",returnRef:projection,
    nodes:[
      node(ctl,"BEHAVIOR","CONTROL","QueueControl."+control.ref,{symbol:semanticIdentity(control),sourcePath:"web/features/queue/behavior.js"}),
      node(state,"BEHAVIOR","STATE","QueueState",{symbol:VexStreamQueueFamily,summary:"empty or positioned Queue truth plus shuffle/repeat flags",sourcePath:"web/features/queue/behavior.js"}),
      node(act,"BEHAVIOR","ACT",actLabel,{sourcePath:"web/features/queue/behavior.js"}),
      node(effect,"BEHAVIOR","EFFECT",effectLabel,{sourcePath:"web/features/queue/behavior.js"}),
      node(projection,"BEHAVIOR","PROJECTION",key+" presentation",{sourcePath:"web/features/queue/presentation.js"}),
      node(method,"DEPTH","METHOD",methodLabel,{summary:"Human listening composition delegates Queue meaning to QueueConnection.",sourcePath:"web/features/player/composition.js"}),
      node(realization,"DEPTH","REALIZATION","QueueConnection.activate()",{symbol:"createQueueConnection",sourcePath:"web/features/queue/connection.js"}),
    ],
    edges:[
      edge(ctl,act,"resolves with current State"),edge(state,act,null,"DATA"),edge(act,effect,"Family resolves"),
      edge(effect,method,"realized through composition"),edge(method,realization),edge(realization,state,"publishes current Queue","RETURN"),edge(state,projection,"projects"),
    ],
  };
}

function previousPathway(){
  const p="pathway.vexstream.player.previous";
  const intent=p+".intent", dispatch=p+".dispatch", playback=p+".playback-state", queue=p+".queue-state", method=p+".method", restart=p+".restart", move=p+".queue-previous", realization=p+".realization";
  return {
    pathwayRef:p,label:"Previous listening action",disposition:"DIRECT_REGISTRATION",
    summary:"The fixed Previous affordance belongs to Player composition, not unconditionally to QueueControl.Previous. Playback time and Queue position choose which already-owned relationship is used.",
    returns:"Current Playback / Queue truth after the selected composition branch",returnRef:realization,
    nodes:[
      node(intent,"BEHAVIOR","INTENT","vexstream.player.previous",{symbol:PREVIOUS_ACTION,sourcePath:"web/index.html"}),
      node(dispatch,"BEHAVIOR","DISPATCH","player.previous()",{summary:"Top-level listening composition.",sourcePath:"web/app.js"}),
      node(playback,"BEHAVIOR","STATE","Current Playback time",{summary:"Elapsed time may make Previous mean restart-current.",sourcePath:"web/features/player/composition.js"}),
      node(queue,"BEHAVIOR","STATE","Current Queue position",{summary:"Near the start, Previous may delegate to QueueControl.Previous when lawful.",sourcePath:"web/features/queue/behavior.js"}),
      node(method,"DEPTH","METHOD","Player.previous()",{sourcePath:"web/features/player/composition.js"}),
      node(restart,"DEPTH","METHOD","Restart current Playback branch",{sourcePath:"web/features/player/composition.js"}),
      node(move,"DEPTH","METHOD","Queue Previous branch",{sourcePath:"web/features/player/composition.js"}),
      node(realization,"DEPTH","REALIZATION","Restart or load current media",{sourcePath:"web/features/player/composition.js"}),
    ],
    edges:[
      edge(intent,dispatch),edge(playback,dispatch,null,"DATA"),edge(queue,dispatch,null,"DATA"),edge(dispatch,method),
      edge(method,restart,"elapsed beyond restart threshold","SIDE"),edge(method,move,"near start with prior Queue item","SIDE"),
      edge(restart,realization),edge(move,realization),
    ],
  };
}

function button(identity,label,pathwayRef,attribute){
  return {ref:elementRef(identity),kind:"BUTTON",label,parentRef:SHELL_REF,symbol:identity,declared:attribute+'="'+identity+'"',intent:identity,pathwayRef,layer:"CHROME",interactions:["ACTIVATE"]};
}

export function buildVexStreamVexBoardLens({html,css}){
  const play=semanticIdentity(PlaybackControl.PlayPause);
  const next=semanticIdentity(QueueControl.Next);
  const shuffle=semanticIdentity(QueueControl.Shuffle);
  const repeat=semanticIdentity(QueueControl.Repeat);
  return {
    schemaVersion:VEXBOARD_SCREEN_LENS_SCHEMA,lensRef:"lens.vexstream-music.desktop",title:"VexStream Music",
    summary:"Bounded projection of the persistent V4 desktop shell and fixed Playback / Queue transport relationships.",
    entryScreenRef:SCREEN_REF,entryElementRef:elementRef(play),relatedBoardRefs:[],
    sourceSnapshot:{repositoryRef:"repo.vexstream-music",repositoryUrl:"https://github.com/vgong24/VexstreamMusic.git",commit:null,observedAt:null,roots:{v2:"v2/"},notes:[
      "Generated by "+VEXSTREAM_VEXBOARD_EXPORTER_VERSION+" from source-owned semantic modules and the lived shell.",
      "Product JavaScript is excluded from the VexBoard static view.",
      "First slice is deliberately bounded to the persistent shell and fixed Playback / Queue transport controls.",
    ]},
    chrome:[],
    screens:[{
      screenRef:SCREEN_REF,label:"VexStream Music desktop",placement:"PRIMARY",symbol:"VexStream V4 lived shell",source:source("web/index.html"),mapped:"MAPPED",
      summary:"Persistent V3-shaped V4 desktop shell. ShellNavigation changes visible terrain while fixed Playback / Queue transport remains available.",
      view:{kind:"STATIC_HTML_CSS",html:viewHTML(html),css:String(css),scriptPolicy:"NONE",htmlSource:source("web/index.html"),cssSource:source("web/styles.css"),anchors:{screenAttribute:"data-vex-screen",elementAttribute:"data-vex-ref",structuralAttribute:"data-vex-structural-ref"},stateBindings:[],interactionBindings:[]},
      elements:[
        {ref:FRAME_REF,kind:"FRAME",label:"Desktop shell frame",parentRef:null,symbol:"body[data-vex-screen]",declared:'data-vex-screen="'+SCREEN_REF+'"'},
        {ref:SHELL_REF,kind:"REGION",label:"V3-shaped V4 shell",parentRef:FRAME_REF,symbol:"main.shell",declared:'<main class="shell">'},
        button(PREVIOUS_ACTION,"Previous","pathway.vexstream.player.previous","data-vex-action"),
        button(play,"Play / Pause","pathway.vexstream.playback.play-pause","data-vex-control"),
        button(next,"Next","pathway.vexstream.queue.next","data-vex-control"),
        button(shuffle,"Shuffle","pathway.vexstream.queue.shuffle","data-vex-control"),
        button(repeat,"Repeat","pathway.vexstream.queue.repeat","data-vex-control"),
      ],
      layout:{kind:"FRAME",symbol:"VexStream desktop shell",source:source("web/index.html"),root:FRAME_REF,frames:[{ref:FRAME_REF,seams:[],regions:[{ref:SHELL_REF,start:"FRAME_START",end:"FRAME_END",top:"FRAME_TOP",bottom:"FRAME_BOTTOM"}],chrome:[]}],overlays:[]},
    }],
    pathways:[
      previousPathway(),playbackPathway(),
      queuePathway("next",QueueControl.Next,QueueAct.Next.ref,QueueEffect.Advance.ref+" / "+QueueEffect.Wrap.ref,"Player.next()"),
      queuePathway("shuffle",QueueControl.Shuffle,QueueAct.EnableShuffle.ref+" / "+QueueAct.DisableShuffle.ref,QueueEffect.ShuffleUpcoming.ref+" / "+QueueEffect.DisableShuffle.ref,"Player.toggleShuffle()"),
      queuePathway("repeat",QueueControl.Repeat,QueueAct.EnableRepeat.ref+" / "+QueueAct.DisableRepeat.ref,QueueEffect.EnableRepeat.ref+" / "+QueueEffect.DisableRepeat.ref,"Player.toggleRepeat()"),
    ],
    annotations:[],examples:[],
    examplesHeld:[{ref:"examples.vexstream-music.dynamic-shell",source:source("tests/player-browser-regression.py"),reason:"The first Lens carries source-default inert HTML/CSS only. Dynamic lived-shell scenarios remain owned by current browser proof until a bounded source-authored expression contract is earned."}],
  };
}

export function readCurrentSources(){
  return {html:fs.readFileSync(path.join(V2,"web","index.html"),"utf8"),css:fs.readFileSync(path.join(V2,"web","styles.css"),"utf8")};
}
export const stableJSON=value=>JSON.stringify(value,null,2)+"\n";

if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)){
  const output=stableJSON(buildVexStreamVexBoardLens(readCurrentSources()));
  if(process.argv.includes("--stdout"))process.stdout.write(output);
  else fs.writeFileSync(path.join(HERE,"vexboard-screen-lens.json"),output);
}
