// Local-only fixture for checking the real Inspector with a native browser.
// It never contacts shops, uses real credentials, or changes production data.
import http from "node:http";
import {readFile} from "node:fs/promises";
import {fileURLToPath} from "node:url";
import path from "node:path";
const root=path.resolve(fileURLToPath(new URL("../dist/",import.meta.url)));
const types={".html":"text/html; charset=utf-8",".js":"application/javascript",".css":"text/css",".svg":"image/svg+xml",".woff2":"font/woff2",".png":"image/png"};
const server=http.createServer(async(req,res)=>{
 const send=(status,data,headers={})=>{res.writeHead(status,{"Content-Type":"application/json",...headers});res.end(JSON.stringify(data));};
 const url=new URL(req.url,"http://127.0.0.1");
 if(url.pathname==="/console/session") return send(200,{id:"fixture",name:"Inspector QA",email:"qa@example.invalid",role:"user",disabled:false,planId:"free",subscriptionStatus:"none"});
 if(req.method==="POST" && url.pathname==="/v1/extract"){
  let body="";for await(const chunk of req){body+=chunk;if(body.length>8192)return send(400,{error:"Fixture input too large"});}
  let target;try{target=new URL(JSON.parse(body).url);}catch{return send(400,{error:"Invalid fixture URL"});}
  if(target.hostname==="limited-store.example") return send(429,{code:"source_rate_limited",error:"Store cooldown fixture"},{"Retry-After":"4"});
  return send(200,{title:"Fixture product",imageUrl:"",description:"Local fixture; no shop contacted.",url:target.href},{"X-Cache":"MISS"});
 }
 try{
  const relative=url.pathname.startsWith("/app") ? "shell.html" : decodeURIComponent(url.pathname.slice(1));
  const file=path.resolve(root,relative);
  if(!file.startsWith(root+path.sep))return send(404,{error:"Not found"});
  const bytes=await readFile(file);res.writeHead(200,{"Content-Type":types[path.extname(file)]||"application/octet-stream","Cache-Control":"no-store"});res.end(bytes);
 }catch{send(404,{error:"Not found"});}
});
server.listen(0,"127.0.0.1",()=>console.log(`Inspector QA: http://127.0.0.1:${server.address().port}/app/inspector`));
