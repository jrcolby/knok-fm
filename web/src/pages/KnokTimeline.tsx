import { useEffect } from "react";
import { useSearchParams } from "react-router";
import { KnokCard } from "../components/KnokCard";
import { useInfiniteKnoks } from "../hooks/useInfiniteKnoks";
import { useIntersectionObserver } from "../hooks/useIntersectionObserver";
import { useScrollBehavior } from "../hooks/useScrollBehavior";
import { useContainerLayout } from "../hooks/useContainerLayout";
import { useQuery } from "@tanstack/react-query";
import { apiClient } from "../api/client";

export function KnokTimeline() {
  const [searchParams] = useSearchParams();
  const searchQuery = searchParams.get("q")?.trim() || "";
  const randomTrigger = searchParams.get("random");

  const isSearchMode = !!searchQuery;
  const isRandomMode = !!randomTrigger;

  // Infinite scroll for timeline and search (global, all servers)
  const {
    knoks,
    isLoading,
    isFetchingNextPage,
    hasNextPage,
    error: infiniteError,
    fetchNextPage,
    refetch,
  } = useInfiniteKnoks({
    searchQuery: isSearchMode ? searchQuery : undefined,
    enabled: !isRandomMode,
  });

  // Separate query for random knok (single result)
  const {
    data: randomData,
    isLoading: isRandomLoading,
    error: randomError,
    refetch: refetchRandom,
  } = useQuery({
    queryKey: ["random-knok", randomTrigger],
    queryFn: () => apiClient.getRandomKnok(),
    enabled: false, // Manual control
    staleTime: 0, // Don't cache random results
  });

  // Intersection observer for infinite scroll trigger
  const loadMoreRef = useIntersectionObserver({
    onIntersect: fetchNextPage,
    enabled: hasNextPage && !isFetchingNextPage && !isRandomMode,
    rootMargin: "200px", // Start loading 200px before reaching the trigger
  });

  // Trigger random fetch when randomTrigger changes
  useEffect(() => {
    if (randomTrigger) {
      refetchRandom();
    }
  }, [randomTrigger, refetchRandom]);

  // Scroll behavior for search/random changes
  useScrollBehavior(searchQuery, randomTrigger);

  // Determine current data and state
  const currentKnoks = isRandomMode ? randomData?.knoks || [] : knoks;
  const currentIsLoading = isRandomMode ? isRandomLoading : isLoading;
  const currentError = isRandomMode ? randomError : infiniteError;
  const hasMoreData = !isRandomMode && hasNextPage;

  // Container layout
  const { containerStyle, loadingErrorStyle } = useContainerLayout(
    currentIsLoading,
    currentError,
    { knoks: currentKnoks, has_more: hasMoreData }
  );

  // Loading state
  if (currentIsLoading && currentKnoks.length === 0) {
    return (
      <div className="relative knok-gradient-bg" style={loadingErrorStyle}>
        <div className="max-w-4xl mx-auto p-8 relative z-10 h-full flex items-center justify-center">
          <div className="animate-pulse">
            <div className="h-8 bg-neutral-700 rounded mb-6"></div>
            <div className="space-y-4">
              {[...Array(3)].map((_, i) => (
                <div key={i} className="bg-neutral-700 h-32 rounded-lg"></div>
              ))}
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Error state
  if (currentError && currentKnoks.length === 0) {
    return (
      <div className="relative knok-gradient-bg" style={loadingErrorStyle}>
        <div className="max-w-4xl mx-auto p-8 relative z-10 h-full flex items-center justify-center">
          <div className="bg-red-900/20 border border-red-800 rounded-lg p-6">
            <h2 className="text-lg font-semibold text-red-400 mb-2">
              {isRandomMode
                ? "Error loading random knok"
                : isSearchMode
                ? "Search error"
                : "Error loading knoks"}
            </h2>
            <p className="text-red-300">
              {currentError instanceof Error
                ? currentError.message
                : "An unknown error occurred"}
            </p>
            <button
              onClick={() => (isRandomMode ? refetchRandom() : refetch())}
              className="mt-4 px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700 transition-colors"
            >
              Try Again
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="relative knok-gradient-bg" style={containerStyle}>
      <div className="max-w-3xl mx-auto p-8 relative z-10">
        {currentKnoks.length === 0 ? (
          <div className="text-center py-12">
            <div className="text-neutral-300 text-lg mb-2">
              {isRandomMode
                ? "No random knok found"
                : isSearchMode
                ? `No results found for "${searchQuery}"`
                : "No knoks found"}
            </div>
            <div className="text-neutral-400">
              {isRandomMode
                ? "Try rolling the dice again"
                : isSearchMode
                ? "Try a different search term"
                : ""}
            </div>
          </div>
        ) : (
          <div className="space-y-6">
            {/* Knok list */}
            {currentKnoks.map((knok) => (
              <KnokCard key={knok.id} knok={knok} />
            ))}

            {/* Infinite scroll loading trigger */}
            {!isRandomMode && hasNextPage && (
              <div
                ref={loadMoreRef}
                className="flex items-center justify-center py-8"
              >
                {isFetchingNextPage ? (
                  <div className="flex items-center gap-2 text-neutral-400">
                    <div className="w-4 h-4 border-2 border-knok-accent border-t-transparent rounded-full animate-spin"></div>
                    <span>Loading more knoks...</span>
                  </div>
                ) : (
                  <div className="text-neutral-500 text-sm">
                    Scroll to load more
                  </div>
                )}
              </div>
            )}

            {/* End of results indicator */}
            {!isRandomMode && !hasNextPage && currentKnoks.length > 0 && (
              <div className="text-center py-8">
                <div className="text-neutral-400">
                  {isSearchMode
                    ? "End of search results"
                    : "You've reached the end"}
                </div>
                <div className="text-xs text-neutral-500 mt-1">
                  {currentKnoks.length} knok
                  {currentKnoks.length !== 1 ? "s" : ""} total
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
