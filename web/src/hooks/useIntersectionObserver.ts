import { useEffect, useRef, useCallback } from "react";

interface UseIntersectionObserverOptions {
  onIntersect: () => void;
  enabled?: boolean;
  rootMargin?: string;
  threshold?: number;
}

/**
 * Hook for observing when an element enters the viewport
 * Perfect for triggering infinite scroll loading
 */
export function useIntersectionObserver({
  onIntersect,
  enabled = true,
  rootMargin = "100px", // Start loading 100px before element is visible
  threshold = 0.1,
}: UseIntersectionObserverOptions) {
  const targetRef = useRef<HTMLDivElement>(null);
  const observerRef = useRef<IntersectionObserver | null>(null);

  const handleIntersect = useCallback(
    (entries: IntersectionObserverEntry[]) => {
      const entry = entries[0];
      if (entry?.isIntersecting && enabled) {
        onIntersect();
      }
    },
    [onIntersect, enabled]
  );

  useEffect(() => {
    const target = targetRef.current;
    if (!target || !enabled) return;

    // Clean up previous observer
    if (observerRef.current) {
      observerRef.current.disconnect();
    }

    // Create new observer
    observerRef.current = new IntersectionObserver(handleIntersect, {
      rootMargin,
      threshold,
    });

    observerRef.current.observe(target);

    return () => {
      if (observerRef.current) {
        observerRef.current.disconnect();
      }
    };
  }, [enabled, handleIntersect, rootMargin, threshold]);

  return targetRef;
}