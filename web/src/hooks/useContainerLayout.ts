import type { KnoksResponse } from "../api/types";

export function useContainerLayout(
  isLoading: boolean, 
  error: unknown, 
  knoksData: KnoksResponse | undefined
) {
  // For single items (random, single search result), prevent overflow
  const isSingleItem = !isLoading && !error && knoksData?.knoks && knoksData.knoks.length === 1;

  // Container styles - let CSS handle overflow naturally
  const getContainerStyle = () => ({
    minHeight: 'calc(100vh - 80px)',
    overflow: isSingleItem ? 'hidden' as const : 'auto' as const
  });

  const getLoadingErrorStyle = () => ({
    height: 'calc(100vh - 80px)',
    overflow: 'hidden' as const
  });

  return {
    isSingleItem,
    containerStyle: getContainerStyle(),
    loadingErrorStyle: getLoadingErrorStyle(),
  };
}