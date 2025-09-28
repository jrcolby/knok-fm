import { useEffect } from "react";

export function useScrollBehavior(searchQuery: string, randomTrigger: string | null) {
  // Effect to scroll to top when random trigger changes
  useEffect(() => {
    if (randomTrigger) {
      window.scrollTo({ top: 0, behavior: 'smooth' });
    }
  }, [randomTrigger]);

  // Effect to scroll to top when search query changes
  useEffect(() => {
    if (searchQuery) {
      window.scrollTo({ top: 0, behavior: 'smooth' });
    }
  }, [searchQuery]);
}