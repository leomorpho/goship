const y=function(){let a=0;return function(){return(++a).toString()}}();function g(a,e="blank",t={icon:{type:"",content:""},duration:"",closeable:!1,theme:{type:"light",style:{background:"",color:"",stroke:""}}}){const o=y(),s=x(o,e,t),r=k(e,t),i=v(a);return s.appendChild(r),s.appendChild(i),t.closeable&&s.appendChild(T(s)),document.querySelector("wc-toast").appendChild(s),{id:o,type:e,message:a,...t}}function x(a,e,t){const{duration:o,theme:s}=t,r=document.createElement("wc-toast-item"),i=(window==null?void 0:window.matchMedia)&&window.matchMedia("(prefers-color-scheme: dark)"),d=i!=null&&i.matches?"dark":"light";if(r.setAttribute("type",e),r.setAttribute("duration",o||""),r.setAttribute("data-toast-item-id",a),r.setAttribute("theme",s!=null&&s.type?s.type:d),(s==null?void 0:s.type)==="custom"&&(s!=null&&s.style)){const{background:m,stroke:c,color:w}=s.style;r.style.setProperty("--wc-toast-background",m),r.style.setProperty("--wc-toast-stroke",c),r.style.setProperty("--wc-toast-color",w)}return r}function k(a,e){const{icon:t}=e,o=document.createElement("wc-toast-icon");return o.setAttribute("type",t!=null&&t.type?t.type:a),o.setAttribute("icon",t!=null&&t.content&&(t==null?void 0:t.type)==="custom"?t.content:""),(t==null?void 0:t.type)==="svg"&&(o.innerHTML=t!=null&&t.content?t.content:""),o}function v(a){const e=document.createElement("wc-toast-content");return e.setAttribute("message",a),e}function T(a){const e=document.createElement("wc-toast-close-button");return e.addEventListener("click",()=>{a.classList.add("dismiss-with-close-button")}),e}function l(a){return function(e,t){return g(e,a,t).id}}function n(a,e){return l("blank")(a,e)}n.loading=l("loading");n.success=l("success");n.error=l("error");n.dismiss=function(a){const e=document.querySelectorAll("wc-toast-item");for(const t of e){const o=t.getAttribute("data-toast-item-id");a===o&&t.classList.add("dismiss")}};n.promise=async function(a,e={loading:"",success:"",error:""},t){const o=n.loading(e.loading,{...t});try{const s=await a;return n.dismiss(o),n.success(e.success,{...t}),s}catch(s){return n.dismiss(o),n.error(e.error,{...t}),s}};class u extends HTMLElement{constructor(){super(),this.attachShadow({mode:"open"}),this.template=document.createElement("template"),this.template.innerHTML=u.template(),this.shadowRoot.append(this.template.content.cloneNode(!0))}connectedCallback(){this.setAttribute("role","status"),this.setAttribute("aria-live","polite"),this.position=this.getAttribute("position")||"top-center",this.arrangeToastPosition(this.position)}static get observedAttributes(){return["position"]}attributeChangedCallback(e,t,o){e==="position"&&(this.position=o,this.arrangeToastPosition(this.position))}arrangeToastPosition(e){const t=e.includes("top"),o={top:t&&0,bottom:!t&&0},s=e.includes("center")?"center":e.includes("right")?"flex-end":"flex-start",r=t?1:-1,i=t?"column-reverse":"column",m=window.getComputedStyle(document.querySelector("html")).getPropertyValue("scrollbar-gutter");this.style.setProperty("--wc-toast-factor",r),this.style.setProperty("--wc-toast-position",s),this.style.setProperty("--wc-toast-direction",i);const c=this.shadowRoot.querySelector(".wc-toast-container");c.style.top=o.top,c.style.bottom=o.bottom,c.style.right=m.includes("stable")&&"4px",c.style.justifyContent=s}static template(){return`
    <style>
      :host {
        --wc-toast-factor: 1;
        --wc-toast-position: center;
        --wc-toast-direction: column-reverse;

        position: fixed;
        z-index: 9999;
        top: 16px;
        left: 16px;
        right: 16px;
        bottom: 16px;
        pointer-events: none;
      }

      .wc-toast-container {
        z-index: 9999;
        left: 0;
        right: 0;
        display: flex;
        position: absolute;
      }

      .wc-toast-wrapper {
        display: flex;
        flex-direction: var(--wc-toast-direction);
        justify-content: flex-end;
        gap: 16px;
        will-change: transform;
        transition: all 230ms cubic-bezier(0.21, 1.02, 0.73, 1);
        pointer-events: none;
      }
    </style>
    <div class="wc-toast-container">
      <div class="wc-toast-wrapper" aria-live="polite">
        <slot> </slot>
      </div>
    </div>
    `}}customElements.define("wc-toast",u);class p extends HTMLElement{constructor(){super(),this.createdAt=new Date,this.EXIT_ANIMATION_DURATION=350,this.attachShadow({mode:"open"}),this.template=document.createElement("template"),this.template.innerHTML=p.template(),this.shadowRoot.append(this.template.content.cloneNode(!0))}connectedCallback(){this.type=this.getAttribute("type")||"blank",this.theme=this.getAttribute("theme")||"light",this.duration=this.getAttribute("duration")||this.getDurationByType(this.type),this.theme==="dark"&&(this.style.setProperty("--wc-toast-background","#2a2a32"),this.style.setProperty("--wc-toast-stroke","#f9f9fa"),this.style.setProperty("--wc-toast-color","#f9f9fa"));const e=()=>{setTimeout(()=>{this.remove()},this.EXIT_ANIMATION_DURATION),this.shadowRoot.querySelector(".wc-toast-bar").classList.add("dismiss")};let t=!1;this.addEventListener("mouseenter",()=>{t=!0}),this.addEventListener("mouseleave",()=>{t=!1});const o=setInterval(()=>{if(this.duration<=0){clearInterval(o),e();return}t||(this.duration-=100)},100)}static get observedAttributes(){return["class"]}attributeChangedCallback(e,t,o){if(e==="class")switch(o){case"dismiss-with-close-button":this.shadowRoot.querySelector(".wc-toast-bar").classList.add("dismiss"),setTimeout(()=>{this.remove()},this.EXIT_ANIMATION_DURATION);break;case"dismiss":default:this.remove();break}}getDurationByType(e){switch(e){case"success":return 2e3;case"loading":return 1e5*60;case"error":case"blank":case"custom":default:return 3500}}static template(){return`
    <style>
      /*
       * Author: Timo Lins
       * License: MIT
       * Source: https://github.com/timolins/react-hot-toast/blob/main/src/components/toast-bar.tsx
       */

      :host {
        --wc-toast-background: #fff;
        --wc-toast-max-width: 350px;
        --wc-toast-stroke: #2a2a32;
        --wc-toast-color: #000;
        --wc-toast-font-family: 'Roboto', 'Amiri', sans-serif;
        --wc-toast-font-size: 16px;
        --wc-toast-border-radius: 8px;
        --wc-toast-content-margin: 4px 10px;

        display: flex;
        justify-content: var(--wc-toast-position);
        transition: all 230ms cubic-bezier(0.21, 1.02, 0.73, 1);
      }

      :host > * {
        pointer-events: auto;
      }

      @media (prefers-color-scheme: dark) {
        :host {
          --wc-toast-background: #2a2a32;
          --wc-toast-stroke: #f9f9fa;
          --wc-toast-color: #f9f9fa;
        }

        :host([theme=light]) {
          --wc-toast-background: #fff;
          --wc-toast-stroke: #2a2a32;
          --wc-toast-color: #000;
        }
      }

      @keyframes enter-animation {
        0% {
          transform: translate3d(0, calc(var(--wc-toast-factor) * -200%), 0) scale(0.6);
          opacity: 0.5;
        }
        100% {
          transform: translate3d(0, 0, 0) scale(1);
          opacity: 1;
        }
      }

      @keyframes exit-animation {
        0% {
          transform: translate3d(0, 0, -1px) scale(1);
          opacity: 1;
        }
        100% {
          transform: translate3d(0, calc(var(--wc-toast-factor) * -150%), -1px) scale(0.6);
          opacity: 0;
        }
      }

      @keyframes fade-in {
        0% {
          opacity: 0;
        }
        100% {
          opacity: 1;
        }
      }

      @keyframes fade-out {
        0% {
          opacity: 1;
        }
        100% {
          opacity: 0;
        }
      }

      .wc-toast-bar {
        display: flex;
        align-items: center;
        background: var(--wc-toast-background, #fff);
        line-height: 1.3;
        will-change: transform;
        box-shadow: 0 3px 10px rgba(0, 0, 0, 0.1), 0 3px 3px rgba(0, 0, 0, 0.05);
        animation: enter-animation 0.3s cubic-bezier(0.21, 1.02, 0.73, 1) forwards;
        max-width: var(--wc-toast-max-width);
        pointer-events: auto;
        padding: 8px 10px;
        border-radius: var(--wc-toast-border-radius);
      }

      .wc-toast-bar.dismiss {
        animation: exit-animation 0.3s forwards cubic-bezier(0.06, 0.71, 0.55, 1);
      }

      @media (prefers-reduced-motion: reduce) {
        .wc-toast-bar {
          animation-name: fade-in;
        }

        .wc-toast-bar.dismiss {
          animation-name: fade-out;
        }
      }
    </style>
    <div class="wc-toast-bar">
      <slot></slot>
    </div>
    `}}customElements.define("wc-toast-item",p);class h extends HTMLElement{constructor(){super(),this.attachShadow({mode:"open"}),this.template=document.createElement("template"),this.template.innerHTML=h.template(),this.shadowRoot.append(this.template.content.cloneNode(!0))}connectedCallback(){this.icon=this.getAttribute("icon"),this.type=this.getAttribute("type")||"blank",this.setAttribute("aria-hidden","true"),this.type!=="svg"&&(this.icon=this.icon!=null?this.createIcon(this.type,this.icon):this.createIcon(this.type),this.shadowRoot.appendChild(this.icon))}createIcon(e="blank",t=""){switch(e){case"success":const o=document.createElement("div");return o.classList.add("checkmark-icon"),o;case"error":const s=document.createElement("div");return s.classList.add("error-icon"),s.innerHTML='<svg focusable="false" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>',s;case"loading":const r=document.createElement("div");return r.classList.add("loading-icon"),r;case"custom":const i=document.createElement("div");return i.classList.add("custom-icon"),i.innerHTML=t,i;case"blank":default:return document.createElement("div")}}static template(){return`
    <style>
      /*
      * Author: Timo Lins
      * License: MIT
      * Source: 
      * - https://github.com/timolins/react-hot-toast/blob/main/src/components/checkmark.tsx
      * - https://github.com/timolins/react-hot-toast/blob/main/src/components/error.tsx
      * - https://github.com/timolins/react-hot-toast/blob/main/src/components/loader.tsx
      */

      :host {
        display: flex;
        align-self: flex-start;
        margin-block: 4px !important;
      }

      @keyframes circle-animation {
        from {
          transform: scale(0) rotate(45deg);
          opacity: 0;
        }
        to {
          transform: scale(1) rotate(45deg);
          opacity: 1;
        }
      }

      .checkmark-icon {
        width: 20px;
        opacity: 0;
        height: 20px;
        border-radius: 10px;
        background: #61d345;
        position: relative;
        transform: rotate(45deg);
        animation: circle-animation 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275) forwards;
        animation-delay: 100ms;
      }

      @keyframes checkmark-animation {
        0% {
          height: 0;
          width: 0;
          opacity: 0;
        }
        40% {
          height: 0;
          width: 6px;
          opacity: 1;
        }
        100% {
          opacity: 1;
          height: 10px;
        }
      }

      .checkmark-icon::after {
        content: '';
        box-sizing: border-box;
        animation: checkmark-animation 0.2s ease-out forwards;
        opacity: 0;
        animation-delay: 200ms;
        position: absolute;
        border-right: 2px solid;
        border-bottom: 2px solid;
        border-color: #fff;
        bottom: 6px;
        left: 6px;
        height: 10px;
        width: 6px;
      }

      @keyframes slide-in {
        from {
          transform: scale(0);
          opacity: 0;
        }
        to {
          transform: scale(1);
          opacity: 1;
        }
      }

      .error-icon {
        width: 20px;
        height: 20px;
        border-radius: 10px;
        background: #ff4b4b;
        display: flex;
        justify-content: center;
        align-items: center;
        animation: slide-in 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275) forwards;
      }

      .error-icon svg{
        width: 16px;
        height: 20px;
        stroke: #fff;
        animation: slide-in .2s ease-out;
        animation-delay: 100ms;
      }

      @keyframes rotate {
        from {
          transform: rotate(0deg);
        }
        to {
          transform: rotate(360deg);
        }
      }

      .loading-icon {
        height: 20px;
        width: 20px;
        position: relative;
        border-radius: 10px;
        background-color: white;
      }

      .loading-icon::after {
        content: '';
        position: absolute;
        bottom: 4px;
        left: 4px;
        width: 12px;
        height: 12px;
        box-sizing: border-box;
        border: 2px solid;
        border-radius: 100%;
        border-color: #e0e0e0;
        border-right-color: #616161;
        animation: rotate 1s linear infinite;
      }

      @media (prefers-color-scheme: dark) {
        ::slotted(svg) {
          stroke: var(--wc-toast-stroke, #fff);
        }
      }
    </style>
    <slot name="svg"></slot>
    `}}customElements.define("wc-toast-icon",h);class f extends HTMLElement{constructor(){super(),this.attachShadow({mode:"open"}),this.template=document.createElement("template"),this.template.innerHTML=f.template(),this.shadowRoot.append(this.template.content.cloneNode(!0))}connectedCallback(){this.message=this.getAttribute("message"),this.shadowRoot.querySelector('slot[name="content"]').innerHTML=this.message}static template(){return`
    <style>
      :host {
        display: flex;
        justify-content: center;
        flex: 1 1 auto;
        margin: var(--wc-toast-content-margin) !important;
        color: var(--wc-toast-color, #000);
        font-family: var(--wc-toast-font-family);
        font-size: var(--wc-toast-font-size);
      }
    </style>
    <slot name="content"></slot>
    `}}customElements.define("wc-toast-content",f);class b extends HTMLElement{constructor(){super(),this.attachShadow({mode:"open"}),this.template=document.createElement("template"),this.template.innerHTML=b.template(),this.shadowRoot.append(this.template.content.cloneNode(!0))}static template(){return`
    <style>
      :host {
        width: 20px;
        opacity: 1;
        height: 20px;
        border-radius: 2px;
        border: 1px solid #dadce0;
        background: var(--wc-toast-background);
        position: relative;
        cursor: pointer;
        display: flex;
        justify-content: center;
        align-items: center;
        margin-left: 5px;
      }

      svg {
        stroke: var(--wc-toast-stroke, #2a2a32);
      }
    </style>
    <svg
      focusable="false"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      stroke="currentColor"
    >
      <path
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="2"
        d="M6 18L18 6M6 6l12 12"
      />
    </svg>
    `}}customElements.define("wc-toast-close-button",b);export{n as t};
//# sourceMappingURL=wc-toast-close-button-BTBYckPC.js.map
