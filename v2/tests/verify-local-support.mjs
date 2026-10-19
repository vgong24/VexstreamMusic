// [VXG RealForever]
// Checkout representation helpers for the repository-owned local verifier.
//
// Git may materialize CRLF on Windows before a candidate .gitattributes file
// becomes active (for example: clone main, then checkout a PR head). Formatting
// proof must ask gofmt about Go syntax/layout, not about that transport artifact.

export function canonicalGoCheckoutText(value){
 return String(value??"").replace(/\r\n/g,"\n");
}
