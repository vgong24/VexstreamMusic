// [VXG RealForever]
// Human-readable overflow motion for track identity text.
//
// The browser reports overflow distance. This owner converts distance to time so
// a longer title gets more time instead of moving faster just to finish inside
// one fixed animation duration.

export const MarqueeReadablePixelsPerSecond=24;
export const MarqueeTravelFraction=.8;
export const MarqueeMinimumSeconds=6.5;

export function marqueeDurationSeconds(
  shiftPixels,
  {
    pixelsPerSecond=MarqueeReadablePixelsPerSecond,
    travelFraction=MarqueeTravelFraction,
    minimumSeconds=MarqueeMinimumSeconds
  }={}
){
  const shift=finiteNonNegative(shiftPixels);
  const speed=finitePositive(pixelsPerSecond);
  const travel=finiteFraction(travelFraction);
  const minimum=finitePositive(minimumSeconds);
  return Math.max(minimum,shift/(speed*travel));
}

function finiteNonNegative(value){
  const number=Number(value);
  return Number.isFinite(number)&&number>0?number:0;
}

function finitePositive(value){
  const number=Number(value);
  if(!Number.isFinite(number)||number<=0){
    throw new Error("marquee timing requires a positive value");
  }
  return number;
}

function finiteFraction(value){
  const number=Number(value);
  if(!Number.isFinite(number)||number<=0||number>1){
    throw new Error("marquee travel fraction must be within (0, 1]");
  }
  return number;
}
