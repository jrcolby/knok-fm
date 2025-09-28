import type { KnokDto } from "../api/types";
import { useState, memo } from "react";
import { KnokSpiral, KnokStar } from "./icons";
import {
  getRandomLogoType,
  needsFallbackLogo,
  LOGO_TYPES,
} from "../utils/logoFallback";

interface KnokCardProps {
  knok: KnokDto;
}

function KnokCardComponent({ knok }: KnokCardProps) {
  const [imageError, setImageError] = useState(false);
  const [imageLoading, setImageLoading] = useState(true);

  // Determine which fallback logo to use (deterministic based on knok ID)
  const logoType = getRandomLogoType(knok.id);
  const shouldUseFallback =
    imageError || needsFallbackLogo(knok.metadata?.image);

  const handleImageError = () => {
    setImageError(true);
    setImageLoading(false);
  };

  const handleImageLoad = () => {
    setImageLoading(false);
  };

  return (
    <article className="bg-knok-bg border border-stone-700 rounded-lg shadow-lg p-4 hover:shadow-xl hover:border-yellow-400/30 transition-all duration-200 h-38">
      <a
        href={knok.url}
        target="_blank"
        rel="noopener noreferrer"
        className="block h-full group"
        aria-label={`Open ${knok.title} in new tab`}
      >
        <div className="flex items-start gap-4 h-full">
          {/* Image container with duotone effect */}
          <div className="relative w-32 h-32 flex-shrink-0 rounded overflow-hidden bg-stone-700 self-center">
            {imageLoading && !shouldUseFallback && (
              <div className="absolute inset-0 bg-stone-600 animate-pulse" />
            )}

            {!shouldUseFallback ? (
              <>
                <img
                  src={knok.metadata.image}
                  alt={`Album art for ${knok.title}`}
                  loading="lazy"
                  className="w-full h-full object-cover grayscale transition-opacity duration-200"
                  onError={handleImageError}
                  onLoad={handleImageLoad}
                  style={{ opacity: imageLoading ? 0 : 1 }}
                />
                {/* Duotone overlay */}
                <div className="absolute inset-0 bg-knok-accent mix-blend-darken opacity-80" />
                <div className="absolute inset-0 bg-knok-bg mix-blend-lighten opacity-60" />
              </>
            ) : (
              // Random logo fallback when no image or error
              <div className="w-full h-full flex items-center justify-center bg-knok-bg">
                {logoType === LOGO_TYPES.SPIRAL ? (
                  <KnokSpiral className="w-16 h-16 text-knok-accent opacity-60" />
                ) : (
                  <KnokStar className="w-16 h-16 text-knok-accent opacity-60" />
                )}
              </div>
            )}
          </div>

          {/* Content */}
          <div className="flex flex-col min-w-0 flex-1">
            <h3 className="text-sm font-semibold text-knok-accent line-clamp-2 group-hover:text-knok-accent/80 transition-colors mb-1 font-plastique">
              {knok.title || "Untitled"}
            </h3>
            {knok.metadata?.description && (
              <p className="text-xs text-stone-300 line-clamp-3 leading-normal break-words overflow-wrap-break-word">
                {knok.metadata.description}
              </p>
            )}
          </div>
        </div>
      </a>
    </article>
  );
}

// Memoize component for better infinite scroll performance
export const KnokCard = memo(KnokCardComponent);
