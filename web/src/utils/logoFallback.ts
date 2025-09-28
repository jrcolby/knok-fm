// Simple deterministic logo fallback utility
// Uses last hex character of UUID for 50/50 split

/**
 * Available logo types for fallback
 */
export const LOGO_TYPES = {
  SPIRAL: 'spiral',
  STAR: 'star',
} as const;

export type LogoType = typeof LOGO_TYPES[keyof typeof LOGO_TYPES];

/**
 * Get deterministic logo type based on last character of knok ID UUID
 * 0-7 = spiral, 8-f = star (50/50 split)
 */
export function getRandomLogoType(knokId: string): LogoType {
  const lastChar = knokId.slice(-1).toLowerCase();
  
  // 0,1,2,3,4,5,6,7 = spiral (8/16 = 50%)
  // 8,9,a,b,c,d,e,f = star (8/16 = 50%)
  return ['0','1','2','3','4','5','6','7'].includes(lastChar) 
    ? LOGO_TYPES.SPIRAL 
    : LOGO_TYPES.STAR;
}

/**
 * Check if a knok needs a fallback logo
 * Returns true if image is missing, empty, or likely to fail
 */
export function needsFallbackLogo(imageUrl?: string): boolean {
  if (!imageUrl) return true;
  if (imageUrl.trim() === '') return true;
  
  // Could add more sophisticated checks here:
  // - URL validation
  // - Known bad patterns
  // - Previous error tracking
  
  return false;
}