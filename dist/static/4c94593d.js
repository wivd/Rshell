import{_ as Ce}from"./2b25fda9.js";/* empty css        *//* empty css        *//* empty css        *//* empty css        *//* empty css        *//* empty css        *//* empty css        *//* empty css        */import"./4ed993c7.js";import{C as se}from"./2a962ec7.js";import{aB as xe,r,e as R,d as be,Z as $e,l as f,m as C,V as l,P as t,p as s,O as Y,U as h,T as c,S as F,J as Te,F as Ve,a8 as Ee,n as ne,az as Se,aA as ze}from"./95bbca75.js";import{a as _,aN as P,v as Ie,z as He,x as Be,w as De,b as ae,y as Ue,f as Re,h as Me,i as Ne,as as Le,aE as Ke,N as Ye,an as Fe,ao as Pe,aO as Oe,O as qe,P as Ae,aP as je}from"./6bd8dd12.js";const i=I=>(Se("data-v-2d0716e2"),I=I(),ze(),I),Ge={class:"note-container"},Je={class:"toolbar-content"},Qe={class:"toolbar-left"},Xe=i(()=>s("h2",{class:"toolbar-title"},[s("i",{class:"el-icon-notebook-2"}),c(" 笔记管理 ")],-1)),Ze={class:"toolbar-info"},We={class:"char-count"},et=i(()=>s("i",{class:"el-icon-document"},null,-1)),tt={class:"line-count"},lt=i(()=>s("i",{class:"el-icon-s-order"},null,-1)),ot={key:0,class:"last-saved"},st=i(()=>s("i",{class:"el-icon-time"},null,-1)),nt={class:"toolbar-right"},at=i(()=>s("i",{class:"el-icon-check"},null,-1)),it=i(()=>s("i",{class:"el-icon-refresh"},null,-1)),ct=i(()=>s("i",{class:"el-icon-download"},null,-1)),ut=i(()=>s("i",{class:"el-icon-arrow-down el-icon--right"},null,-1)),dt=i(()=>s("i",{class:"el-icon-document"},null,-1)),rt=i(()=>s("i",{class:"el-icon-edit"},null,-1)),_t=i(()=>s("i",{class:"el-icon-link"},null,-1)),mt=i(()=>s("i",{class:"el-icon-collection-tag"},null,-1)),pt=i(()=>s("i",{class:"el-icon-arrow-down el-icon--right"},null,-1)),vt=i(()=>s("i",{class:"el-icon-s-flag"},null,-1)),ft=i(()=>s("i",{class:"el-icon-warning"},null,-1)),ht=i(()=>s("i",{class:"el-icon-finished"},null,-1)),yt=i(()=>s("i",{class:"el-icon-setting"},null,-1)),gt={class:"editor-header"},wt=i(()=>s("span",{class:"editor-title"},[s("i",{class:"el-icon-edit"}),c(" 笔记内容 ")],-1)),kt={class:"editor-tools"},Ct={key:0,class:"fullscreen-editor"},xt={class:"fullscreen-tools"},bt={key:1},$t={class:"editor-wrapper"},Tt={key:0,class:"quick-tools"},Vt=i(()=>s("strong",null,"B",-1)),Et=i(()=>s("em",null,"I",-1)),St=i(()=>s("code",null,"`",-1)),zt=i(()=>s("span",{class:"custom-icon"},"•",-1)),It=i(()=>s("span",{class:"custom-icon"},"1.",-1)),Ht=i(()=>s("span",{class:"custom-icon"},">",-1)),Bt=i(()=>s("span",{class:"custom-icon"},"—",-1)),Dt=i(()=>s("span",{class:"custom-icon"},"🔗",-1)),Ut=i(()=>s("span",{class:"custom-icon"},"🖼️",-1)),Rt={key:0,class:"preview-area"},Mt={class:"preview-header"},Nt=i(()=>s("span",null,"预览",-1)),Lt=i(()=>s("i",{class:"el-icon-close"},null,-1)),Kt=["innerHTML"],Yt={class:"status-bar"},Ft={class:"status-left"},Pt=i(()=>s("i",{class:"el-icon-warning"},null,-1)),Ot=i(()=>s("i",{class:"el-icon-success"},null,-1)),qt={class:"cursor-position"},At={class:"status-right"},jt=i(()=>s("i",{class:"el-icon-view"},null,-1)),Gt=i(()=>s("i",{class:"el-icon-menu"},null,-1)),Jt=i(()=>s("i",{class:"el-icon-time"},null,-1)),Qt={class:"history-header"},Xt=i(()=>s("i",{class:"el-icon-refresh-left"},null,-1)),Zt={class:"history-content"},Wt={__name:"ClientNotes",setup(I){const O=xe().query.uid,u=r(""),H=r(!1),y=r(!1),x=r(!1),B=r(!0),M=r(!1),b=r(!1),$=r(!1),g=r(null),q=r(1),A=r(1),N=r(""),T=r([]),V=r(""),v=r({name:"",content:""}),ie=R(()=>u.value.length),ce=R(()=>u.value.split(`
`).length),E=R(()=>u.value!==V.value),j=R(()=>{let o=u.value;return o=o.replace(/\*\*(.*?)\*\*/g,"<strong>$1</strong>"),o=o.replace(/\*(.*?)\*/g,"<em>$1</em>"),o=o.replace(/`(.*?)`/g,"<code>$1</code>"),o=o.replace(/^# (.*$)/gm,"<h1>$1</h1>"),o=o.replace(/^## (.*$)/gm,"<h2>$1</h2>"),o=o.replace(/^### (.*$)/gm,"<h3>$1</h3>"),o=o.replace(/^- (.*$)/gm,"<li>$1</li>"),o=o.replace(/^> (.*$)/gm,"<blockquote>$1</blockquote>"),o=o.replace(/---/g,"<hr>"),o=o.replace(/\[(.*?)\]\((.*?)\)/g,'<a href="$2">$1</a>'),o=o.replace(/\!\[(.*?)\]\((.*?)\)/g,'<img src="$2" alt="$1">'),o=o.replace(/\n/g,"<br>"),o}),ue=async()=>{var o;try{const e=await se.get_note({uid:O});((o=e==null?void 0:e.data)==null?void 0:o.data)!=null?(u.value=e.data.data,V.value=e.data.data,Z("初始加载")):(u.value="",V.value="")}catch(e){_.error("加载笔记失败")}},D=async()=>{if(!E.value){_.info("内容未修改，无需保存");return}H.value=!0;try{(await se.save_note({uid:O,noteContent:u.value})).data.status===200?(V.value=u.value,N.value=P().format("HH:mm:ss"),Z("手动保存"),_.success("笔记已保存")):_.error("保存失败")}catch(o){_.error("保存笔记失败")}finally{H.value=!1}},de=async()=>{try{await ae.confirm("确定要重置笔记吗？未保存的修改将丢失。","确认重置",{confirmButtonText:"确定",cancelButtonText:"取消",type:"warning"}),u.value=V.value,_.success("已重置")}catch{}},G=()=>{M.value&&E.value&&fe()},J=o=>{if((o.ctrlKey||o.metaKey)&&o.key==="s"&&(o.preventDefault(),D()),(o.ctrlKey||o.metaKey)&&o.key==="e"&&(o.preventDefault(),X("txt")),(o.ctrlKey||o.metaKey)&&o.key==="f"&&(o.preventDefault(),Q()),g.value){const e=g.value.$el.querySelector("textarea");if(e){const n=e.selectionStart,d=u.value.substring(0,n);q.value=d.split(`
`).length,A.value=d.length-d.lastIndexOf(`
`)}}},m=o=>{if(g.value){const e=g.value.$el.querySelector("textarea");if(e){const n=e.selectionStart,d=e.selectionEnd,p=u.value.substring(n,d);p?u.value=u.value.substring(0,n)+o.replace("文字",p)+u.value.substring(d):u.value=u.value.substring(0,n)+o+u.value.substring(n),ne(()=>{e.focus();const w=n+o.length;e.setSelectionRange(w,w)})}}},Q=()=>{y.value=!y.value,y.value&&ne(()=>{g.value&&g.value.focus()})},X=async o=>{const e={txt:"text/plain",md:"text/markdown",html:"text/html"};let n=u.value,d=`note_${P().format("YYYYMMDD_HHmmss")}`;o==="html"?(n=`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>笔记导出</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        h1 { color: #333; }
        code { background: #f5f5f5; padding: 2px 4px; }
    </style>
</head>
<body>
    <div>${j.value}</div>
</body>
</html>`,d+=".html"):o==="md"?d+=".md":d+=".txt";const p=new Blob([n],{type:e[o]||"text/plain"}),w=URL.createObjectURL(p),k=document.createElement("a");k.href=w,k.download=d,document.body.appendChild(k),k.click(),document.body.removeChild(k),URL.revokeObjectURL(w),_.success(`已导出为${o.toUpperCase()}格式`)},re=async o=>{const e={pentest:`# 渗透测试笔记

## 目标信息
- 目标名称：
- IP地址：
- 端口信息：

## 信息收集
- 开放端口：
- 服务版本：
- 目录扫描：

## 漏洞发现

## 利用过程

## 权限提升

## 痕迹清理

## 总结建议`,vuln:`# 漏洞报告

## 漏洞信息
- 漏洞名称：
- 风险等级：
- CVSS评分：

## 漏洞描述

## 影响范围

## 复现步骤
1.
2.
3.

## 修复建议

## 参考链接`,report:`# 渗透测试报告

## 执行摘要

## 测试范围

## 测试方法

## 发现漏洞
### 高风险
### 中风险
### 低风险

## 修复建议

## 附录
- 测试工具
- 测试时间
- 测试人员`};if(o==="custom")$.value=!0;else if(e[o])try{await ae.confirm("使用模板将替换当前内容，确定要继续吗？","确认使用模板",{confirmButtonText:"确定",cancelButtonText:"取消",type:"warning"}),u.value=e[o],_.success("模板已应用")}catch{}},_e=()=>{if(!v.value.name){_.error("请输入模板名称");return}if(!v.value.content){_.error("请输入模板内容");return}localStorage.setItem(`note_template_${v.value.name}`,v.value.content),$.value=!1,v.value={name:"",content:""},_.success("模板已保存")},Z=o=>{T.value.unshift({time:P().format("YYYY-MM-DD HH:mm:ss"),content:u.value,preview:u.value.substring(0,100)+(u.value.length>100?"...":""),action:o}),T.value.length>10&&T.value.pop()},me=()=>{b.value=!0},pe=o=>{u.value=o,_.success("已恢复历史版本"),b.value=!1},ve=()=>{T.value=[],_.success("历史记录已清空")};let S=null;const fe=()=>{S&&clearTimeout(S),S=setTimeout(async()=>{await D(),_.success("已自动保存")},2e3)},W=o=>{y.value&&o.key==="Escape"&&(y.value=!1)};return be(()=>{ue(),window.addEventListener("keydown",W)}),$e(()=>{window.removeEventListener("keydown",W),S&&clearTimeout(S)}),(o,e)=>{const n=Ie,d=Ue,p=Re,w=Me,k=Ne,z=Le,L=He,he=Ke,U=Ye,K=Fe,ye=Pe,ee=Be,ge=je,we=Oe,te=De,le=qe,ke=Ae;return f(),C("div",Ge,[l(L,{shadow:"never",class:"toolbar-card"},{default:t(()=>[s("div",Je,[s("div",Qe,[Xe,s("div",Ze,[s("span",We,[et,c(" 字数："+h(ie.value),1)]),s("span",tt,[lt,c(" 行数："+h(ce.value),1)]),N.value?(f(),C("span",ot,[st,c(" 上次保存："+h(N.value),1)])):F("",!0)])]),s("div",nt,[l(z,{class:"action-buttons"},{default:t(()=>[l(d,{content:"保存 (Ctrl+S)",placement:"top"},{default:t(()=>[l(n,{type:"primary",onClick:D,loading:H.value,disabled:!E.value},{default:t(()=>[at,c(" 保存 ")]),_:1},8,["loading","disabled"])]),_:1}),l(d,{content:"重置",placement:"top"},{default:t(()=>[l(n,{onClick:de,disabled:!E.value},{default:t(()=>[it,c(" 重置 ")]),_:1},8,["disabled"])]),_:1}),l(k,{onCommand:X},{dropdown:t(()=>[l(w,null,{default:t(()=>[l(p,{command:"txt"},{default:t(()=>[dt,c(" TXT格式 ")]),_:1}),l(p,{command:"md"},{default:t(()=>[rt,c(" Markdown格式 ")]),_:1}),l(p,{command:"html"},{default:t(()=>[_t,c(" HTML格式 ")]),_:1})]),_:1})]),default:t(()=>[l(n,null,{default:t(()=>[ct,c(" 导出 "),ut]),_:1})]),_:1})]),_:1}),l(k,{trigger:"click",onCommand:re},{dropdown:t(()=>[l(w,null,{default:t(()=>[l(p,{command:"pentest"},{default:t(()=>[vt,c(" 渗透测试模板 ")]),_:1}),l(p,{command:"vuln"},{default:t(()=>[ft,c(" 漏洞报告模板 ")]),_:1}),l(p,{command:"report"},{default:t(()=>[ht,c(" 测试报告模板 ")]),_:1}),l(p,{command:"custom",divided:""},{default:t(()=>[yt,c(" 自定义模板 ")]),_:1})]),_:1})]),default:t(()=>[l(n,{type:"info"},{default:t(()=>[mt,c(" 模板 "),pt]),_:1})]),_:1})])])]),_:1}),l(L,{shadow:"never",class:"editor-card"},{header:t(()=>[s("div",gt,[wt,s("div",kt,[l(d,{content:"自动保存",placement:"top"},{default:t(()=>[l(he,{modelValue:M.value,"onUpdate:modelValue":e[0]||(e[0]=a=>M.value=a),"active-text":"自动保存","inactive-text":"手动保存",size:"small"},null,8,["modelValue"])]),_:1}),l(d,{content:"全屏编辑",placement:"top"},{default:t(()=>[l(n,{type:"text",size:"small",onClick:Q,class:"fullscreen-btn"},{default:t(()=>[s("i",{class:Te(y.value?"el-icon-close":"el-icon-full-screen")},null,2)]),_:1})]),_:1})])])]),default:t(()=>[y.value?(f(),C("div",Ct,[l(U,{ref_key:"editorRef",ref:g,modelValue:u.value,"onUpdate:modelValue":e[1]||(e[1]=a=>u.value=a),type:"textarea",autosize:{minRows:20},placeholder:"在这里输入笔记...",class:"fullscreen-textarea",onInput:G,onKeydown:J},null,8,["modelValue"]),s("div",xt,[l(n,{type:"primary",onClick:D,loading:H.value},{default:t(()=>[c(" 保存并退出 ")]),_:1},8,["loading"]),l(n,{onClick:e[2]||(e[2]=a=>y.value=!1)},{default:t(()=>[c(" 退出全屏 ")]),_:1})])])):(f(),C("div",bt,[s("div",$t,[l(U,{ref_key:"editorRef",ref:g,modelValue:u.value,"onUpdate:modelValue":e[3]||(e[3]=a=>u.value=a),type:"textarea",autosize:{minRows:25,maxRows:25},placeholder:"在这里输入笔记内容...",class:"note-textarea",onInput:G,onKeydown:J},null,8,["modelValue"]),B.value?(f(),C("div",Tt,[l(ye,null,{default:t(()=>[l(z,{size:"small"},{default:t(()=>[l(n,{onClick:e[4]||(e[4]=a=>m("**粗体文字**"))},{default:t(()=>[Vt]),_:1}),l(n,{onClick:e[5]||(e[5]=a=>m("*斜体文字*"))},{default:t(()=>[Et]),_:1}),l(n,{onClick:e[6]||(e[6]=a=>m("`代码片段`"))},{default:t(()=>[St]),_:1})]),_:1}),l(K,{direction:"vertical"}),l(z,{size:"small"},{default:t(()=>[l(n,{onClick:e[7]||(e[7]=a=>m("# 标题"))},{default:t(()=>[c(" H1 ")]),_:1}),l(n,{onClick:e[8]||(e[8]=a=>m("## 标题"))},{default:t(()=>[c(" H2 ")]),_:1}),l(n,{onClick:e[9]||(e[9]=a=>m("### 标题"))},{default:t(()=>[c(" H3 ")]),_:1})]),_:1}),l(K,{direction:"vertical"}),l(z,{size:"small"},{default:t(()=>[l(n,{onClick:e[10]||(e[10]=a=>m("- 列表项"))},{default:t(()=>[zt]),_:1}),l(n,{onClick:e[11]||(e[11]=a=>m("1. 有序项"))},{default:t(()=>[It]),_:1}),l(n,{onClick:e[12]||(e[12]=a=>m("> 引用内容"))},{default:t(()=>[Ht]),_:1})]),_:1}),l(K,{direction:"vertical"}),l(z,{size:"small"},{default:t(()=>[l(n,{onClick:e[13]||(e[13]=a=>m("---"))},{default:t(()=>[Bt]),_:1}),l(n,{onClick:e[14]||(e[14]=a=>m("[链接](https://)"))},{default:t(()=>[Dt]),_:1}),l(n,{onClick:e[15]||(e[15]=a=>m("![图片](url)"))},{default:t(()=>[Ut]),_:1})]),_:1})]),_:1})])):F("",!0)]),x.value?(f(),C("div",Rt,[s("div",Mt,[Nt,l(n,{type:"text",size:"small",onClick:e[16]||(e[16]=a=>x.value=!1)},{default:t(()=>[Lt]),_:1})]),s("div",{class:"preview-content",innerHTML:j.value},null,8,Kt)])):F("",!0)]))]),_:1}),s("div",Yt,[s("div",Ft,[E.value?(f(),Y(ee,{key:0,type:"warning",size:"small",class:"dirty-tag"},{default:t(()=>[Pt,c(" 未保存 ")]),_:1})):(f(),Y(ee,{key:1,type:"success",size:"small",class:"saved-tag"},{default:t(()=>[Ot,c(" 已保存 ")]),_:1})),s("span",qt," 第 "+h(q.value)+" 行, 第 "+h(A.value)+" 列 ",1)]),s("div",At,[l(n,{type:"text",size:"small",onClick:e[17]||(e[17]=a=>x.value=!x.value)},{default:t(()=>[jt,c(" "+h(x.value?"隐藏预览":"预览"),1)]),_:1}),l(n,{type:"text",size:"small",onClick:e[18]||(e[18]=a=>B.value=!B.value)},{default:t(()=>[Gt,c(" "+h(B.value?"隐藏工具栏":"显示工具栏"),1)]),_:1}),l(n,{type:"text",size:"small",onClick:me},{default:t(()=>[Jt,c(" 历史记录 ")]),_:1})])]),l(te,{modelValue:b.value,"onUpdate:modelValue":e[20]||(e[20]=a=>b.value=a),title:"笔记历史记录",width:"800px"},{footer:t(()=>[l(n,{onClick:e[19]||(e[19]=a=>b.value=!1)},{default:t(()=>[c("关闭")]),_:1}),l(n,{type:"primary",onClick:ve},{default:t(()=>[c("清空历史记录")]),_:1})]),default:t(()=>[l(we,null,{default:t(()=>[(f(!0),C(Ve,null,Ee(T.value,(a,oe)=>(f(),Y(ge,{key:oe,timestamp:a.time,type:oe===0?"primary":"",placement:"top"},{default:t(()=>[l(L,{shadow:"never"},{header:t(()=>[s("div",Qt,[s("span",null,h(a.time),1),l(n,{size:"small",type:"text",onClick:tl=>pe(a.content)},{default:t(()=>[Xt,c(" 恢复此版本 ")]),_:2},1032,["onClick"])])]),default:t(()=>[s("div",Zt,h(a.preview),1)]),_:2},1024)]),_:2},1032,["timestamp","type"]))),128))]),_:1})]),_:1},8,["modelValue"]),l(te,{modelValue:$.value,"onUpdate:modelValue":e[24]||(e[24]=a=>$.value=a),title:"自定义模板",width:"600px"},{footer:t(()=>[l(n,{onClick:e[23]||(e[23]=a=>$.value=!1)},{default:t(()=>[c("取消")]),_:1}),l(n,{type:"primary",onClick:_e},{default:t(()=>[c("保存模板")]),_:1})]),default:t(()=>[l(ke,{model:v.value,"label-width":"100px"},{default:t(()=>[l(le,{label:"模板名称"},{default:t(()=>[l(U,{modelValue:v.value.name,"onUpdate:modelValue":e[21]||(e[21]=a=>v.value.name=a),placeholder:"请输入模板名称"},null,8,["modelValue"])]),_:1}),l(le,{label:"模板内容"},{default:t(()=>[l(U,{modelValue:v.value.content,"onUpdate:modelValue":e[22]||(e[22]=a=>v.value.content=a),type:"textarea",rows:10,placeholder:"请输入模板内容"},null,8,["modelValue"])]),_:1})]),_:1},8,["model"])]),_:1},8,["modelValue"])])}}},vl=Ce(Wt,[["__scopeId","data-v-2d0716e2"]]);export{vl as default};
