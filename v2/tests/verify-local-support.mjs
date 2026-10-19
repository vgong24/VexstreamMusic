// [VXG RealForever]
// Checkout representation helpers for the repository-owned local verifier.
//
// Git may materialize CRLF on Windows before a candidate .gitattributes file
// becomes active (for example: clone main, then checkout a PR head). Formatting
// proof must ask gofmt about Go syntax/layout, not about that transport artifact.

export function canonicalGoCheckoutText(value){
 return String(value??"").replace(/\r\n/g,"\n");
}

function quoteWindowsCommandToken(value){
 const text=String(value);
 if(/[&|<>^()%!\r\n]/.test(text)){
  throw new Error("unsafe Windows command token: "+text);
 }
 if(!/[\s"]/g.test(text))return text;
 return `"${text
  .replace(/(\\*)"/g,"$1$1\\\"")
  .replace(/(\\+)$/,"$1$1")}"`;
}

export function physicalLocalCommand(
 command,
 args=[],
 {
  platform=process.platform,
  comspec=process.env.ComSpec||
   process.env.COMSPEC||
   "C:\\Windows\\System32\\cmd.exe"
 }={}
){
 const argv=[...args].map(value=>String(value));
 if(platform!=="win32"||!/\.(cmd|bat)$/i.test(command)){
  return Object.freeze({command,args:Object.freeze(argv)});
 }
 const line=[command,...argv].map(quoteWindowsCommandToken).join(" ");
 return Object.freeze({
  command:comspec,
  args:Object.freeze(["/d","/s","/c",line])
 });
}
