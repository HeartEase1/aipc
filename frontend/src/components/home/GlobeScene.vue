<template>
  <div ref="wrapperRef" class="globe-scene" :class="{ 'is-interacting': isUserInteracting }">
    <!-- WebGL canvas -->
    <canvas
      v-show="!webglUnavailable"
      ref="canvasRef"
      class="globe-canvas"
      :aria-label="t('home.interactiveGlobe')"
    />

    <div v-if="webglUnavailable" class="globe-fallback" role="status">
      {{ t('home.globeUnavailable') }}
    </div>

    <button
      v-if="!webglUnavailable"
      type="button"
      class="globe-reset"
      data-testid="globe-reset"
      :aria-label="t('home.resetGlobeView')"
      :title="t('home.resetGlobeView')"
      @click.stop="resetView"
    >
      <Icon name="refresh" size="sm" :stroke-width="2" />
    </button>

    <!-- AI logo orbit (CSS overlay, 3D→2D projected each frame) -->
    <div v-if="!webglUnavailable" class="layer-orbit">
      <div
        v-for="logo in logoStates"
        :key="logo.id"
        class="logo-pill"
        :data-orbit="logo.orbitId"
        :data-provider="logo.providerId"
        :style="{
          transform: `translate(calc(${logo.x}px - 50%), calc(${logo.y}px - 50%)) scale(${logo.scale})`,
          opacity: logo.opacity,
          zIndex: logo.zIndex,
        }"
      >
        <svg viewBox="0 0 24 24" class="pill-icon">
          <path v-for="(d, i) in logo.paths" :key="i" :d="d" :fill="logo.color" />
        </svg>
        <span class="pill-name">{{ logo.name }}</span>
      </div>
    </div>

    <!-- Node labels (project with globe rotation each frame) -->
    <div v-if="!webglUnavailable" class="layer-nodes">
      <div
        v-for="nd in nodeStates"
        :key="nd.id"
        class="node-label"
        :style="{
          transform: `translate(calc(${nd.x}px - 50%), calc(${nd.y}px - 50%))`,
          opacity: nd.opacity,
        }"
      >
        <span class="node-pulse" :style="{ '--c': nd.color }" />
        <span class="node-id" :style="{ color: nd.color }">{{ nd.id }}</span>
        <span class="node-region">{{ t(nd.regionKey) }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import * as THREE from 'three'
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

// ─── constants ────────────────────────────────────────────────────────────────
const R = 1.0 // globe radius
const MIN_CAMERA_DISTANCE = 2.7
const MAX_CAMERA_DISTANCE = 5.4
const INTRO_CAMERA_Z = 5.2
const AUTO_ROTATE_RESUME_DELAY_MS = 3000
const HOME_CAMERA_POSITION = new THREE.Vector3(0, 0.3, 3.55)

// ─── node definitions ─────────────────────────────────────────────────────────
const NODE_DEFS = [
  { id: 'US', lat:  37.7, lon:  -96.0, color: '#60a5fa', regionKey: 'home.nodes.us' },
  { id: 'HK', lat:  22.3, lon:  114.2, color: '#34d399', regionKey: 'home.nodes.hk' },
  { id: 'SG', lat:   1.4, lon:  103.8, color: '#a78bfa', regionKey: 'home.nodes.sg' },
] as const

// ─── AI logo definitions (SVG paths from ModelIcon.vue) ───────────────────────
const LOGO_DEFS = [
  { id:'openai', name:'ChatGPT', color:'#10a37f',
    paths:['M21.55 10.004a5.416 5.416 0 00-.478-4.501c-1.217-2.09-3.662-3.166-6.05-2.66A5.59 5.59 0 0010.831 1C8.39.995 6.224 2.546 5.473 4.838A5.553 5.553 0 001.76 7.496a5.487 5.487 0 00.691 6.5 5.416 5.416 0 00.477 4.502c1.217 2.09 3.662 3.165 6.05 2.66A5.586 5.586 0 0013.168 23c2.443.006 4.61-1.546 5.361-3.84a5.553 5.553 0 003.715-2.66 5.488 5.488 0 00-.693-6.497v.001zm-8.381 11.558a4.199 4.199 0 01-2.675-.954c.034-.018.093-.05.132-.074l4.44-2.53a.71.71 0 00.364-.623v-6.176l1.877 1.069c.02.01.033.029.036.05v5.115c-.003 2.274-1.87 4.118-4.174 4.123zM4.192 17.78a4.059 4.059 0 01-.498-2.763c.032.02.09.055.131.078l4.44 2.53c.225.13.504.13.73 0l5.42-3.088v2.138a.068.068 0 01-.027.057L9.9 19.288c-1.999 1.136-4.552.46-5.707-1.51h-.001zM3.023 8.216A4.15 4.15 0 015.198 6.41l-.002.151v5.06a.711.711 0 00.364.624l5.42 3.087-1.876 1.07a.067.067 0 01-.063.005l-4.489-2.559c-1.995-1.14-2.679-3.658-1.53-5.63h.001zm15.417 3.54l-5.42-3.088L14.896 7.6a.067.067 0 01.063-.006l4.489 2.557c1.998 1.14 2.683 3.662 1.529 5.633a4.163 4.163 0 01-2.174 1.807V12.38a.71.71 0 00-.363-.623zm1.867-2.773a6.04 6.04 0 00-.132-.078l-4.44-2.53a.731.731 0 00-.729 0l-5.42 3.088V7.325a.068.068 0 01.027-.057L14.1 4.713c2-1.137 4.555-.46 5.707 1.513.487.833.664 1.809.499 2.757h.001zm-11.741 3.81l-1.877-1.068a.065.065 0 01-.036-.051V6.559c.001-2.277 1.873-4.122 4.181-4.12.976 0 1.92.338 2.671.954-.034.018-.092.05-.131.073l-4.44 2.53a.71.71 0 00-.365.623l-.003 6.173v.002zm1.02-2.168L12 9.25l2.414 1.375v2.75L12 14.75l-2.415-1.375v-2.75z'] },
  { id:'gemini', name:'Gemini',  color:'#4285f4',
    paths:['M20.616 10.835a14.147 14.147 0 01-4.45-3.001 14.111 14.111 0 01-3.678-6.452.503.503 0 00-.975 0 14.134 14.134 0 01-3.679 6.452 14.155 14.155 0 01-4.45 3.001c-.65.28-1.318.505-2.002.678a.502.502 0 000 .975c.684.172 1.35.397 2.002.677a14.147 14.147 0 014.45 3.001 14.112 14.112 0 013.679 6.453.502.502 0 00.975 0c.172-.685.397-1.351.677-2.003a14.145 14.145 0 013.001-4.45 14.113 14.113 0 016.453-3.678.503.503 0 000-.975 13.245 13.245 0 01-2.003-.678z'] },
  { id:'xai',    name:'Grok',    color:'#e2e8f0',
    paths:['M9.27 15.29l7.978-5.897c.391-.29.95-.177 1.137.272.98 2.369.542 5.215-1.41 7.169-1.951 1.954-4.667 2.382-7.149 1.406l-2.711 1.257c3.889 2.661 8.611 2.003 11.562-.953 2.341-2.344 3.066-5.539 2.388-8.42l.006.007c-.983-4.232.242-5.924 2.75-9.383.06-.082.12-.164.179-.248l-3.301 3.305v-.01L9.267 15.292M7.623 16.723c-2.792-2.67-2.31-6.801.071-9.184 1.761-1.763 4.647-2.483 7.166-1.425l2.705-1.25a7.808 7.808 0 00-1.829-1A8.975 8.975 0 005.984 5.83c-2.533 2.536-3.33 6.436-1.962 9.764 1.022 2.487-.653 4.246-2.34 6.022-.599.63-1.199 1.259-1.682 1.925l7.62-6.815'] },
  { id:'claude', name:'Claude',  color:'#d97706',
    paths:['M4.709 15.955l4.72-2.647.08-.23-.08-.128H9.2l-.79-.048-2.698-.073-2.339-.097-2.266-.122-.571-.121L0 11.784l.055-.352.48-.321.686.06 1.52.103 2.278.158 1.652.097 2.449.255h.389l.055-.157-.134-.098-.103-.097-2.358-1.596-2.552-1.688-1.336-.972-.724-.491-.364-.462-.158-1.008.656-.722.881.06.225.061.893.686 1.908 1.476 2.491 1.833.365.304.145-.103.019-.073-.164-.274-1.355-2.446-1.446-2.49-.644-1.032-.17-.619a2.97 2.97 0 01-.104-.729L6.283.134 6.696 0l.996.134.42.364.62 1.414 1.002 2.229 1.555 3.03.456.898.243.832.091.255h.158V9.01l.128-1.706.237-2.095.23-2.695.08-.76.376-.91.747-.492.584.28.48.685-.067.444-.286 1.851-.559 2.903-.364 1.942h.212l.243-.242.985-1.306 1.652-2.064.73-.82.85-.904.547-.431h1.033l.76 1.129-.34 1.166-1.064 1.347-.881 1.142-1.264 1.7-.79 1.36.073.11.188-.02 2.856-.606 1.543-.28 1.841-.315.833.388.091.395-.328.807-1.969.486-2.309.462-3.439.813-.042.03.049.061 1.549.146.662.036h1.622l3.02.225.79.522.474.638-.079.485-1.215.62-1.64-.389-3.829-.91-1.312-.329h-.182v.11l1.093 1.068 2.006 1.81 2.509 2.33.127.578-.322.455-.34-.049-2.205-1.657-.851-.747-1.926-1.62h-.128v.17l.444.649 2.345 3.521.122 1.08-.17.353-.608.213-.668-.122-1.374-1.925-1.415-2.167-1.143-1.943-.14.08-.674 7.254-.316.37-.729.28-.607-.461-.322-.747.322-1.476.389-1.924.315-1.53.286-1.9.17-.632-.012-.042-.14.018-1.434 1.967-2.18 2.945-1.726 1.845-.414.164-.717-.37.067-.662.401-.589 2.388-3.036 1.44-1.882.93-1.086-.006-.158h-.055L4.132 18.56l-1.13.146-.487-.456.061-.746.231-.243 1.908-1.312-.006.006z'] },
] as const

type LogoOrbitDefinition = {
  id: 'primary' | 'cross'
  radius: number
  tilt: number
  roll: number
  speed: number
  phase: number
  ringOpacity: number
  order: readonly number[]
}

const LOGO_ORBITS = [
  {
    id: 'primary',
    radius: 1.45,
    tilt: 0.42,
    roll: 0.64,
    speed: 0.26,
    phase: 0,
    ringOpacity: 0.35,
    order: [0, 1, 2, 3],
  },
  {
    id: 'cross',
    radius: 1.52,
    tilt: 0.42,
    roll: -0.64,
    speed: -0.19,
    phase: Math.PI * 0.35,
    ringOpacity: 0.28,
    order: [3, 2, 1, 0],
  },
] as const satisfies readonly LogoOrbitDefinition[]

const LOGO_ORBIT_PITCH = 0.34

// ─── reactive overlay state ────────────────────────────────────────────────────
type LogoState = { id:string; providerId:string; orbitId:string; orbitIndex:number; name:string; color:string; angle:number; paths:string[]; x:number; y:number; scale:number; opacity:number; zIndex:number }
type NodeState = { id:string; lat:number; lon:number; color:string; regionKey:string; x:number; y:number; opacity:number }

const logoStates = reactive<LogoState[]>(LOGO_ORBITS.flatMap((orbit, orbitIndex) =>
  orbit.order.map((logoIndex, position) => {
    const logo = LOGO_DEFS[logoIndex]
    return {
      ...logo,
      id: `${orbit.id}-${logo.id}`,
      providerId: logo.id,
      orbitId: orbit.id,
      orbitIndex,
      angle: position * Math.PI * 0.5,
      paths: [...logo.paths] as string[],
      x: 0,
      y: 0,
      scale: 1,
      opacity: 0,
      zIndex: 1,
    }
  }),
))
const nodeStates = reactive<NodeState[]>(NODE_DEFS.map(n => ({ ...n, x:0, y:0, opacity:0 })))

// ─── refs & internals ─────────────────────────────────────────────────────────
const wrapperRef = ref<HTMLDivElement|null>(null)
const canvasRef  = ref<HTMLCanvasElement|null>(null)
const webglUnavailable = ref(false)
const isUserInteracting = ref(false)

let renderer:   THREE.WebGLRenderer     | null = null
let scene:      THREE.Scene             | null = null
let camera:     THREE.PerspectiveCamera | null = null
let globeGroup: THREE.Group             | null = null
let controls:   OrbitControls            | null = null
let atmosphereMaterial: THREE.ShaderMaterial | null = null
let rafId:      number                  | null = null
let destroyed = false
let pageVisible = true
let isIntersecting = true
let reducedMotion = false
let lastFrameTime = 0
let elapsedTime = 0
let resizeObserver: ResizeObserver | null = null
let intersectionObserver: IntersectionObserver | null = null
let motionQuery: MediaQueryList | null = null
let autoRotateResumeAt = 0
let lastCameraDistanceData = ''
const nodeLocs: THREE.Vector3[] = []

type ArcFlow = { curve:THREE.CatmullRomCurve3; attr:THREE.BufferAttribute; t:Float32Array; v:Float32Array }
const flows: ArcFlow[] = []
const arcMats: THREE.ShaderMaterial[] = []   // for time uniform update
let   introT = 0                             // 0→1 intro camera animation

// ─── traffic particle system ──────────────────────────────────────────────────
const TRAFFIC_N  = 22
const TRAIL_LEN  = 14
// Node colors as [R,G,B] 0-1 — matches NODE_DEFS colors
const NODE_RGB = [[0.376,0.647,0.980],[0.204,0.827,0.600],[0.655,0.471,0.980]]

type TParticle = { spawn:THREE.Vector3; target:number; t:number; speed:number; trail:THREE.Vector3[] }
const tParticles: TParticle[] = []
let tPosAttr:   THREE.BufferAttribute | null = null
let tAlphaAttr: THREE.BufferAttribute | null = null
let tColAttr:   THREE.BufferAttribute | null = null

function randSurface(): THREE.Vector3 {
  return ll2v((Math.random()-0.5)*160, Math.random()*360-180)
}
function nearestNode(p: THREE.Vector3): number {
  let best=0, bd=Infinity
  nodeLocs.forEach((n,i)=>{ const d=p.distanceTo(n); if(d<bd){bd=d;best=i} })
  return best
}
function slerp3(a: THREE.Vector3, b: THREE.Vector3, t: number): THREE.Vector3 {
  const dot=Math.max(-1,Math.min(1,a.dot(b))), ang=Math.acos(dot)
  if(ang<0.0001) return a.clone().lerp(b,t)
  const s=Math.sin(ang)
  return a.clone().multiplyScalar(Math.sin((1-t)*ang)/s).add(b.clone().multiplyScalar(Math.sin(t*ang)/s))
}
function spawnTP(p: TParticle){ p.spawn=randSurface(); p.target=nearestNode(p.spawn); p.t=0; p.speed=0.13+Math.random()*0.24; p.trail=[] }


// ─── helper ───────────────────────────────────────────────────────────────────
function ll2v(lat:number, lon:number, r=R): THREE.Vector3 {
  const phi   = (90-lat)  * Math.PI/180
  const theta = (lon+180) * Math.PI/180
  return new THREE.Vector3(
    -r * Math.sin(phi) * Math.cos(theta),
     r * Math.cos(phi),
     r * Math.sin(phi) * Math.sin(theta))
}

const ATMO_VERT = `
  varying vec3 vNormal; varying vec3 vWorldPos;
  void main(){
    vNormal   = normalize(normalMatrix*normal);
    vWorldPos = (modelMatrix*vec4(position,1.0)).xyz;
    gl_Position = projectionMatrix*modelViewMatrix*vec4(position,1.0);
  }`
const ATMO_FRAG = `
  varying vec3 vNormal; varying vec3 vWorldPos;
  uniform vec3 uCam;
  void main(){
    float f = 1.0 - abs(dot(normalize(vNormal), normalize(uCam-vWorldPos)));
    f = pow(f,2.8);
    gl_FragColor = vec4(0.25,0.55,1.0, f*0.75);
  }`

// Arc animated-dash shader — uses tube UV to draw moving dashes with glow
const ARC_VERT = `varying vec2 vUv; void main(){ vUv=uv; gl_Position=projectionMatrix*modelViewMatrix*vec4(position,1.0); }`
const ARC_FRAG = `
  uniform float time; uniform vec3 arcColor;
  varying vec2 vUv;
  void main(){
    // Dashes moving along arc length (vUv.x 0→1)
    float t   = fract(vUv.x * 7.0 - time * 0.55);
    float dash = smoothstep(0.0,0.10,t) * smoothstep(0.55,0.40,t);
    // Radial glow across tube cross-section (vUv.y 0→1)
    float r   = abs(vUv.y - 0.5) * 2.0;
    float rim = 1.0 - r * r;
    gl_FragColor = vec4(arcColor, dash * rim * 0.92);
  }`

// ─── scene builder ────────────────────────────────────────────────────────────
function buildGlobe(): THREE.Group {
  const g = new THREE.Group()

  // Stars background
  const starPos = new Float32Array(6000)
  for (let i=0;i<6000;i++) {
    const th=Math.random()*Math.PI*2, ph=Math.acos(Math.random()*2-1)
    const d=80+Math.random()*20
    starPos[i*3]  =d*Math.sin(ph)*Math.cos(th)
    starPos[i*3+1]=d*Math.cos(ph)
    starPos[i*3+2]=d*Math.sin(ph)*Math.sin(th)
  }
  const sg=new THREE.BufferGeometry(); sg.setAttribute('position',new THREE.BufferAttribute(starPos,3))
  scene!.add(new THREE.Points(sg, new THREE.PointsMaterial({color:0xffffff,size:0.18,transparent:true,opacity:0.75})))

  // Earth sphere – real texture from /earth.jpg
  const loader = new THREE.TextureLoader()
  const mat = new THREE.MeshPhongMaterial({shininess:28,specular:new THREE.Color(0x224466)})
  loader.load('/earth.jpg', tex => {
    if (destroyed) {
      tex.dispose()
      return
    }
    mat.map=tex
    mat.needsUpdate=true
    renderCurrentFrame()
  })
  g.add(new THREE.Mesh(new THREE.SphereGeometry(R,72,36), mat))

  // Atmosphere rim (custom Fresnel shader)
  const cam3 = camera!.position.clone()
  atmosphereMaterial = new THREE.ShaderMaterial({
    vertexShader: ATMO_VERT, fragmentShader: ATMO_FRAG,
    uniforms:{ uCam:{value:cam3} },
    side:THREE.BackSide, transparent:true, blending:THREE.AdditiveBlending, depthWrite:false
  })
  g.add(new THREE.Mesh(new THREE.SphereGeometry(R+0.065,48,24), atmosphereMaterial))

  // Node markers
  nodeLocs.length = 0
  for (const nd of NODE_DEFS) {
    const pos = ll2v(nd.lat, nd.lon)
    const col = new THREE.Color(nd.color)
    nodeLocs.push(pos.clone())
    const dot = new THREE.Mesh(new THREE.SphereGeometry(0.020,12,8), new THREE.MeshBasicMaterial({color:col}))
    dot.position.copy(pos); g.add(dot)
    const outward = pos.clone().normalize()
    for (let ri=0;ri<2;ri++) {
      const rm = new THREE.Mesh(
        new THREE.RingGeometry(0.032+ri*0.022, 0.044+ri*0.022,24),
        new THREE.MeshBasicMaterial({color:col,transparent:true,opacity:0.4-ri*0.15,side:THREE.DoubleSide}))
      rm.position.copy(pos); rm.quaternion.setFromUnitVectors(new THREE.Vector3(0,0,1),outward); g.add(rm)
    }
  }

  // Arc tubes: slerp-based, surface-hugging (peak lift 0.18), fix near-antipodal via waypoint
  const vecs = NODE_DEFS.map(n=>ll2v(n.lat,n.lon))
  const ARC_H = 0.18  // low elevation — arcs hug globe surface

  function buildArcPts(n1: THREE.Vector3, n2: THREE.Vector3): THREE.Vector3[] {
    const dot12 = n1.dot(n2)
    const pts: THREE.Vector3[] = []
    if (dot12 < -0.75) {
      // Near-antipodal (e.g. US↔SG): route via Pacific midpoint (20°N,175°E)
      const via = ll2v(20, 175).normalize()
      for (let s=0;s<=16;s++){
        const f=s/16
        pts.push(slerp3(n1,via,f).multiplyScalar(R + Math.sin(f*Math.PI)*ARC_H))
      }
      for (let s=1;s<=16;s++){
        const f=s/16
        pts.push(slerp3(via,n2,f).multiplyScalar(R + Math.sin(f*Math.PI)*ARC_H))
      }
    } else {
      for (let s=0;s<=32;s++){
        const f=s/32
        pts.push(slerp3(n1,n2,f).multiplyScalar(R + Math.sin(f*Math.PI)*ARC_H))
      }
    }
    return pts
  }

  // All 3 pairs — bidirectional
  const pairs: [number,number,string][] = [
    [0,1,'#38bdf8'], // US↔HK
    [1,2,'#34d399'], // HK↔SG
    [2,0,'#a78bfa'], // SG↔US (near-antipodal, routed via Pacific)
  ]
  for (const [i,j,hex] of pairs) {
    const n1=vecs[i].clone().normalize(), n2=vecs[j].clone().normalize()
    const pts = buildArcPts(n1, n2)
    const curve = new THREE.CatmullRomCurve3(pts, false, 'catmullrom', 0.3)
    const col = new THREE.Color(hex)
    const arcMat = new THREE.ShaderMaterial({
      vertexShader: ARC_VERT, fragmentShader: ARC_FRAG,
      uniforms:{ time:{value:0}, arcColor:{value:col} },
      transparent:true, depthWrite:false, blending:THREE.AdditiveBlending, side:THREE.DoubleSide
    })
    arcMats.push(arcMat)
    g.add(new THREE.Mesh(new THREE.TubeGeometry(curve,100,0.003,8,false), arcMat))
    const N=12, pos3=new Float32Array(N*3)
    const attr=new THREE.BufferAttribute(pos3,3)
    const pg=new THREE.BufferGeometry(); pg.setAttribute('position',attr)
    g.add(new THREE.Points(pg, new THREE.PointsMaterial({
      color: new THREE.Color(hex), size:0.038, transparent:true, opacity:0.95,
      sizeAttenuation:true, blending:THREE.AdditiveBlending, depthWrite:false
    })))
    flows.push({ curve, attr,
      t: Float32Array.from({length:N},(_,k) => k/N),
      v: Float32Array.from({length:N},(_,k) => {
        const speed = 0.07 + Math.random()*0.08
        return k < N/2 ? speed : -speed
      })
    })
  }
  return g
}

// ─── orbit ring (rendered in scene, not globeGroup — doesn't rotate) ──────────
function pointOnLogoOrbit(orbit: LogoOrbitDefinition, angle: number): THREE.Vector3 {
  const baseX=orbit.radius*Math.cos(angle)
  const baseY=orbit.radius*Math.sin(angle)*Math.sin(orbit.tilt)
  const baseZ=orbit.radius*Math.sin(angle)*Math.cos(orbit.tilt)
  const cosRoll=Math.cos(orbit.roll)
  const sinRoll=Math.sin(orbit.roll)
  const rolledX=baseX*cosRoll-baseY*sinRoll
  const rolledY=baseX*sinRoll+baseY*cosRoll
  const cosPitch=Math.cos(LOGO_ORBIT_PITCH)
  const sinPitch=Math.sin(LOGO_ORBIT_PITCH)
  return new THREE.Vector3(
    rolledX,
    rolledY*cosPitch-baseZ*sinPitch,
    rolledY*sinPitch+baseZ*cosPitch,
  )
}

function addOrbitRings(){
  for(const orbit of LOGO_ORBITS){
    const pts: THREE.Vector3[] = []
    for(let i=0;i<=96;i++){
      pts.push(pointOnLogoOrbit(orbit,i/96*Math.PI*2))
    }
    scene!.add(new THREE.Line(
      new THREE.BufferGeometry().setFromPoints(pts),
      new THREE.LineBasicMaterial({color:0x1e3a5f, transparent:true, opacity:orbit.ringOpacity})
    ))
  }
}

// ─── sphere occlusion: returns true if worldPos is hidden behind the globe ────
function isBehindGlobe(wp: THREE.Vector3, cam: THREE.Vector3): boolean {
  const dir = wp.clone().sub(cam)
  const len = dir.length()
  dir.normalize()
  // Project globe center (origin) onto ray
  const oc = cam.clone().negate()
  const tca = oc.dot(dir)
  if(tca < 0) return false
  const d2 = oc.dot(oc) - tca*tca
  if(d2 > R*R) return false
  return (tca - Math.sqrt(R*R - d2)) < len
}

// ─── per-frame ────────────────────────────────────────────────────────────────
function tickFlow(dt:number){
  for(const f of flows){
    for(let k=0;k<f.t.length;k++){
      // (+1)%1 handles negative velocities correctly for backward particles
      f.t[k] = ((f.t[k] + f.v[k]*dt) % 1 + 1) % 1
      const p=f.curve.getPoint(f.t[k]); f.attr.setXYZ(k,p.x,p.y,p.z)
    }
    f.attr.needsUpdate=true
  }
}

function buildTrafficSystem(group: THREE.Group){
  const total = TRAFFIC_N * TRAIL_LEN
  const pos   = new Float32Array(total*3)
  const alpha = new Float32Array(total)
  const col   = new Float32Array(total*3)
  const posA  = new THREE.BufferAttribute(pos,3)
  const alpA  = new THREE.BufferAttribute(alpha,1)
  const colA  = new THREE.BufferAttribute(col,3)
  const geo = new THREE.BufferGeometry()
  geo.setAttribute('position',posA); geo.setAttribute('alpha',alpA); geo.setAttribute('vcolor',colA)
  const mat = new THREE.ShaderMaterial({
    vertexShader:`
      attribute float alpha; attribute vec3 vcolor;
      varying float vA; varying vec3 vC;
      void main(){
        vA=alpha; vC=vcolor;
        gl_PointSize=max(1.2, 5.5*pow(alpha,0.6));
        gl_Position=projectionMatrix*modelViewMatrix*vec4(position,1.0);
      }`,
    fragmentShader:`
      varying float vA; varying vec3 vC;
      void main(){
        vec2 c=gl_PointCoord-0.5; float d=length(c);
        if(d>0.5)discard;
        gl_FragColor=vec4(vC, vA*(1.0-d*1.7));
      }`,
    transparent:true, depthWrite:false, blending:THREE.AdditiveBlending
  })
  group.add(new THREE.Points(geo,mat))
  tPosAttr=posA; tAlphaAttr=alpA; tColAttr=colA
  for(let i=0;i<TRAFFIC_N;i++){
    const p:TParticle={spawn:new THREE.Vector3(),target:0,t:Math.random(),speed:0,trail:[]}
    spawnTP(p); p.t=Math.random(); tParticles.push(p)
  }
}

function tickTraffic(dt:number){
  if(!tPosAttr||!tAlphaAttr||!tColAttr||!nodeLocs.length)return
  for(let i=0;i<TRAFFIC_N;i++){
    const p=tParticles[i]
    p.t+=p.speed*dt
    if(p.t>=1){ spawnTP(p); continue }
    const cur=slerp3(p.spawn,nodeLocs[p.target],p.t).normalize().multiplyScalar(R+0.012)
    p.trail.unshift(cur.clone())
    if(p.trail.length>TRAIL_LEN) p.trail.length=TRAIL_LEN
    const rgb=NODE_RGB[p.target]
    for(let j=0;j<TRAIL_LEN;j++){
      const idx=i*TRAIL_LEN+j
      const tp=j<p.trail.length?p.trail[j]:p.spawn
      tPosAttr.setXYZ(idx,tp.x,tp.y,tp.z)
      tAlphaAttr.setX(idx,Math.max(0, 1-j/(TRAIL_LEN-1)*1.1))
      tColAttr.setXYZ(idx,rgb[0],rgb[1],rgb[2])
    }
  }
  tPosAttr.needsUpdate=true; tAlphaAttr.needsUpdate=true; tColAttr.needsUpdate=true
}

function tickLogos(elapsed:number){
  if(!camera||!wrapperRef.value)return
  const W=wrapperRef.value.clientWidth, H=wrapperRef.value.clientHeight||480
  for(let i=0;i<logoStates.length;i++){
    const logo=logoStates[i]
    const orbit=LOGO_ORBITS[logo.orbitIndex]
    const a=logo.angle+orbit.phase+elapsed*orbit.speed
    const wp=pointOnLogoOrbit(orbit,a)
    const v=wp.clone().project(camera)
    logo.x=(v.x*0.5+0.5)*W
    logo.y=(-v.y*0.5+0.5)*H

    // Accurate sphere occlusion
    const occluded = isBehindGlobe(wp, camera.position)
    const depth=THREE.MathUtils.clamp(wp.z/orbit.radius,-1,1)
    const depthRatio=(depth+1)/2
    const targetOpacity = occluded ? 0 : (0.50+depthRatio*0.50)
    const targetScale   = occluded ? 0.62 : (0.68+depthRatio*0.42)
    const targetZ       = occluded ? -1  : Math.round(((depth+1)/2)*20)

    // Lerp for silky-smooth transitions (no abrupt snapping)
    const lf = 0.12   // lerp factor per frame (~60fps → ~5 frames to transition)
    logo.opacity += (targetOpacity-logo.opacity)*lf
    logo.scale   += (targetScale-logo.scale)*lf
    logo.zIndex  = targetZ
  }
}

function tickNodes(){
  if(!camera||!globeGroup||!wrapperRef.value)return
  const W=wrapperRef.value.clientWidth, H=wrapperRef.value.clientHeight||480
  for(let i=0;i<nodeLocs.length;i++){
    const world=nodeLocs[i].clone().applyMatrix4(globeGroup.matrixWorld)
    const camDir=world.clone().sub(camera.position).normalize()
    const facing=world.clone().normalize().dot(camDir)
    nodeStates[i].opacity = facing<-0.1 ? Math.min(1,(-facing-0.1)*5) : 0
    const v=world.clone().project(camera)
    nodeStates[i].x=(v.x*0.5+0.5)*W
    nodeStates[i].y=(-v.y*0.5+0.5)*H
  }
}

// ─── loop ─────────────────────────────────────────────────────────────────────
function easeOutExpo(x: number){ return x>=1 ? 1 : 1-Math.pow(2,-10*x) }

function syncCameraDependentState(){
  controls?.update()
  updateCameraDistanceData()
  if(camera&&atmosphereMaterial){
    atmosphereMaterial.uniforms.uCam.value.copy(camera.position)
  }
}

function updateFrame(dt:number, elapsed:number, now:number){
  if(!renderer||!scene||!camera||!globeGroup)return
  // ── Intro animation: camera zoom-in over 2.2 s ──
  if(introT < 1){
    introT = Math.min(1, introT + dt/2.2)
    const e = easeOutExpo(introT)
    camera.position.z = INTRO_CAMERA_Z + (HOME_CAMERA_POSITION.z-INTRO_CAMERA_Z)*e
    globeGroup.scale.setScalar(0.75 + 0.25*e)
  }

  syncCameraDependentState()

  // ── Globe rotation (paused while the user is manipulating the camera) ──
  if(!isUserInteracting.value&&now>=autoRotateResumeAt){
    globeGroup.rotation.y += dt * 0.12 * Math.min(1, introT*3)
  }

  // ── Arc shader time update ──
  arcMats.forEach(m => { m.uniforms.time.value = elapsed })

  tickFlow(dt); tickTraffic(dt); tickLogos(elapsed); tickNodes()
  renderer.render(scene,camera)
}

function renderCurrentFrame(){
  if(destroyed||!renderer||!scene||!camera||!globeGroup)return
  syncCameraDependentState()
  tickFlow(0); tickTraffic(0); tickLogos(elapsedTime); tickNodes()
  renderer.render(scene,camera)
}

function shouldAnimate(){
  return !destroyed && !webglUnavailable.value && pageVisible && isIntersecting && !reducedMotion
}

function loop(now:number){
  rafId=null
  if(!shouldAnimate())return
  const dt=Math.min(Math.max((now-lastFrameTime)/1000,0),0.1)
  lastFrameTime=now
  elapsedTime+=dt
  updateFrame(dt,elapsedTime,now)
  rafId=requestAnimationFrame(loop)
}

function startLoop(){
  if(rafId!==null||!shouldAnimate())return
  lastFrameTime=performance.now()
  rafId=requestAnimationFrame(loop)
}

function stopLoop(){
  if(rafId!==null)cancelAnimationFrame(rafId)
  rafId=null
  lastFrameTime=0
}

function updateAnimationState(){
  if(reducedMotion){
    stopLoop()
    introT=1
    camera?.position.copy(HOME_CAMERA_POSITION)
    globeGroup?.scale.setScalar(1)
    renderCurrentFrame()
    return
  }
  if(shouldAnimate())startLoop()
  else stopLoop()
}

function onControlsStart(){
  isUserInteracting.value=true
  introT=1
  globeGroup?.scale.setScalar(1)
  autoRotateResumeAt=Number.POSITIVE_INFINITY
}

function onControlsEnd(){
  isUserInteracting.value=false
  autoRotateResumeAt=performance.now()+AUTO_ROTATE_RESUME_DELAY_MS
}

function updateCameraDistanceData(){
  if(!camera||!controls||!wrapperRef.value)return
  const nextDistance=camera.position.distanceTo(controls.target).toFixed(3)
  if(nextDistance===lastCameraDistanceData)return
  lastCameraDistanceData=nextDistance
  wrapperRef.value.dataset.cameraDistance=nextDistance
}

function onControlsChange(){
  updateCameraDistanceData()
  if(reducedMotion&&renderer&&scene&&camera&&globeGroup){
    if(atmosphereMaterial)atmosphereMaterial.uniforms.uCam.value.copy(camera.position)
    tickLogos(elapsedTime)
    tickNodes()
    renderer.render(scene,camera)
  }
}

function setupControls(){
  if(!camera||!canvasRef.value)return
  controls=new OrbitControls(camera,canvasRef.value)
  controls.enableDamping=true
  controls.dampingFactor=0.07
  controls.enablePan=false
  controls.enableZoom=true
  controls.rotateSpeed=0.55
  controls.zoomSpeed=0.8
  controls.minDistance=MIN_CAMERA_DISTANCE
  controls.maxDistance=MAX_CAMERA_DISTANCE
  controls.minPolarAngle=0.12
  controls.maxPolarAngle=Math.PI-0.12
  controls.target.set(0,0,0)
  controls.addEventListener('start',onControlsStart)
  controls.addEventListener('end',onControlsEnd)
  controls.addEventListener('change',onControlsChange)
  controls.update()
  updateCameraDistanceData()
}

function resetView(){
  if(!camera||!controls||!globeGroup)return
  introT=1
  camera.position.copy(HOME_CAMERA_POSITION)
  controls.target.set(0,0,0)
  controls.update()
  globeGroup.position.set(0,0,0)
  globeGroup.rotation.set(0,0,0)
  globeGroup.scale.setScalar(1)
  isUserInteracting.value=false
  autoRotateResumeAt=performance.now()+AUTO_ROTATE_RESUME_DELAY_MS
  renderCurrentFrame()
}

function onVisibilityChange(){
  pageVisible=!document.hidden
  updateAnimationState()
}

function onMotionPreferenceChange(event:MediaQueryListEvent){
  reducedMotion=event.matches
  updateAnimationState()
}

function setupObservers(){
  pageVisible=!document.hidden
  document.addEventListener('visibilitychange',onVisibilityChange)

  if(typeof ResizeObserver!=='undefined'&&wrapperRef.value){
    resizeObserver=new ResizeObserver(onResize)
    resizeObserver.observe(wrapperRef.value)
  }else{
    window.addEventListener('resize',onResize)
  }

  if(typeof IntersectionObserver!=='undefined'&&wrapperRef.value){
    intersectionObserver=new IntersectionObserver(entries=>{
      isIntersecting=entries[0]?.isIntersecting??false
      updateAnimationState()
    },{threshold:0.01})
    intersectionObserver.observe(wrapperRef.value)
  }

  if(typeof window.matchMedia==='function'){
    motionQuery=window.matchMedia('(prefers-reduced-motion: reduce)')
    reducedMotion=motionQuery.matches
    motionQuery.addEventListener?.('change',onMotionPreferenceChange)
  }
}

function teardownObservers(){
  document.removeEventListener('visibilitychange',onVisibilityChange)
  resizeObserver?.disconnect()
  resizeObserver=null
  intersectionObserver?.disconnect()
  intersectionObserver=null
  window.removeEventListener('resize',onResize)
  motionQuery?.removeEventListener?.('change',onMotionPreferenceChange)
  motionQuery=null
}

function disposeSceneResources(){
  controls?.removeEventListener('start',onControlsStart)
  controls?.removeEventListener('end',onControlsEnd)
  controls?.removeEventListener('change',onControlsChange)
  controls?.dispose()
  controls=null
  lastCameraDistanceData=''
  scene?.traverse(object=>{
    const renderable=object as THREE.Object3D & { geometry?:THREE.BufferGeometry; material?:THREE.Material|THREE.Material[] }
    renderable.geometry?.dispose()
    const materials=Array.isArray(renderable.material)?renderable.material:[renderable.material]
    materials.forEach(material=>{
      if(!material)return
      Object.values(material).forEach(value=>{
        if(value instanceof THREE.Texture)value.dispose()
      })
      material.dispose()
    })
  })
  scene?.clear()
  renderer?.renderLists.dispose()
  renderer?.dispose()
  renderer=null
  scene=null
  camera=null
  globeGroup=null
  atmosphereMaterial=null
}

// ─── init ─────────────────────────────────────────────────────────────────────
function init(){
  if(!canvasRef.value||!wrapperRef.value)return
  try{
    const W=wrapperRef.value.clientWidth, H=wrapperRef.value.clientHeight||480
    scene  = new THREE.Scene()
    camera = new THREE.PerspectiveCamera(42,W/H,0.1,200)
    camera.position.set(0,0.3,INTRO_CAMERA_Z); camera.lookAt(0,0,0)
    renderer=new THREE.WebGLRenderer({canvas:canvasRef.value,alpha:true,antialias:true})
    renderer.setSize(W,H); renderer.setPixelRatio(Math.min(devicePixelRatio,2))
    renderer.setClearColor(0x000000,0)
    scene.add(new THREE.AmbientLight(0xffffff,0.4))
    const sun=new THREE.DirectionalLight(0xfff4e0,1.2); sun.position.set(5,3,5); scene.add(sun)
    globeGroup=buildGlobe()
    buildTrafficSystem(globeGroup)
    scene.add(globeGroup)
    addOrbitRings()
    setupControls()
    setupObservers()
    updateAnimationState()
  }catch(error){
    console.warn('[home-globe] WebGL initialization failed:',error)
    destroyed=true
    webglUnavailable.value=true
    stopLoop()
    teardownObservers()
    disposeSceneResources()
  }
}

function onResize(){
  if(!renderer||!camera||!wrapperRef.value)return
  const W=wrapperRef.value.clientWidth, H=wrapperRef.value.clientHeight||480
  camera.aspect=W/H; camera.updateProjectionMatrix(); renderer.setSize(W,H)
  renderCurrentFrame()
}

onMounted(init)
onUnmounted(()=>{
  destroyed=true
  stopLoop()
  teardownObservers()
  disposeSceneResources()
  flows.length=0; tParticles.length=0; arcMats.length=0; nodeLocs.length=0
  tPosAttr=null; tAlphaAttr=null; tColAttr=null
})
</script>

<style scoped>
.globe-scene {
  position: relative;
  width: 100%;
  height: 400px;
  min-height: 400px;
  overflow: hidden;
  isolation: isolate;
}

@media (min-width: 640px) and (max-width: 1023px) {
  .globe-scene {
    height: 480px;
    min-height: 480px;
  }
}

/* 桌面端使用稳定高度，避免 ResizeObserver 与 Canvas 内联尺寸互相放大。 */
@media (min-width: 1024px) {
  .globe-scene {
    height: 620px;
    min-height: 620px;
  }
}

.globe-canvas {
  display: block;
  width: 100%;
  height: 100%;
  cursor: grab;
  touch-action: none;
}

.globe-scene.is-interacting .globe-canvas {
  cursor: grabbing;
}

.globe-reset {
  position: absolute;
  top: 16px;
  right: 16px;
  z-index: 30;
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  border: 1px solid rgba(255,255,255,0.14);
  border-radius: 8px;
  color: rgba(255,255,255,0.72);
  background: rgba(5,10,22,0.72);
  box-shadow: 0 8px 24px rgba(0,0,0,0.25);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  transition: color 0.18s ease, border-color 0.18s ease, background 0.18s ease;
}

.globe-reset:hover {
  color: white;
  border-color: rgba(56,189,248,0.46);
  background: rgba(14,28,50,0.86);
}

.globe-reset:focus-visible {
  outline: 2px solid rgba(56,189,248,0.9);
  outline-offset: 2px;
}

.globe-fallback {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  padding-top: 250px;
  color: rgba(255, 255, 255, 0.46);
  font-size: 12px;
  text-align: center;
}

.globe-fallback::before {
  content: '';
  position: absolute;
  width: min(58vw, 280px);
  aspect-ratio: 1;
  border-radius: 50%;
  background: url('/earth.jpg') center / cover no-repeat;
  opacity: 0.5;
  box-shadow: 0 0 70px rgba(56, 189, 248, 0.2);
}

/* ── Logo orbit overlay ── */
.layer-orbit {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.logo-pill {
  position: absolute;
  top: 0;
  left: 0;
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 6px 13px 6px 8px;
  border-radius: 999px;
  /* NO transition on transform (set every frame by JS) — only opacity gets CSS ease */
  background: rgba(8, 14, 28, 0.72);
  border: 1px solid rgba(255, 255, 255, 0.12);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.4), inset 0 1px 0 rgba(255,255,255,0.07);
  white-space: nowrap;
  pointer-events: none;
  transition: opacity 0.18s ease;
  will-change: transform, opacity;
}

.pill-icon {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

.pill-name {
  font-size: 12px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.9);
  letter-spacing: 0.02em;
}

/* ── Node labels overlay ── */
.layer-nodes {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.node-label {
  position: absolute;
  top: 0;
  left: 0;
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 3px 9px 3px 5px;
  border-radius: 6px;
  background: rgba(5, 10, 22, 0.75);
  border: 1px solid rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  white-space: nowrap;
  pointer-events: none;
  transition: opacity 0.25s ease;
  will-change: transform, opacity;
}

/* Pulsing dot via CSS animation */
.node-pulse {
  position: relative;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--c);
  box-shadow: 0 0 6px var(--c);
  flex-shrink: 0;
}
.node-pulse::after {
  content: '';
  position: absolute;
  inset: -3px;
  border-radius: 50%;
  border: 1px solid var(--c);
  opacity: 0;
  animation: pulse-ring 2s ease-out infinite;
}

@keyframes pulse-ring {
  0%   { transform: scale(1);   opacity: 0.8; }
  100% { transform: scale(2.2); opacity: 0;   }
}

@media (prefers-reduced-motion: reduce) {
  .node-pulse::after { animation: none; }
}

.node-id {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.05em;
}

.node-region {
  font-size: 10px;
  color: rgba(255, 255, 255, 0.42);
}
</style>
