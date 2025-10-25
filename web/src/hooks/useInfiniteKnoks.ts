import { useInfiniteQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { apiClient } from "../api/client";
import type { KnokDto } from "../api/types";

interface UseInfiniteKnoksOptions {
  searchQuery?: string;
  enabled?: boolean;
}

interface UseInfiniteKnoksResult {
  knoks: KnokDto[];
  isLoading: boolean;
  isFetchingNextPage: boolean;
  hasNextPage: boolean;
  error: Error | null;
  fetchNextPage: () => void;
  refetch: () => void;
}

export function useInfiniteKnoks({
  searchQuery,
  enabled = true,
}: UseInfiniteKnoksOptions): UseInfiniteKnoksResult {
  const [isTyping, setIsTyping] = useState(false);
  
  const isSearchMode = !!searchQuery;

  // Infinite query for timeline knoks (global, all servers)
  const timelineQuery = useInfiniteQuery({
    queryKey: ["infinite-knoks"], // No serverId - global timeline
    queryFn: ({ pageParam }) =>
      apiClient.getKnoks(pageParam, 25), // Global endpoint
    enabled: enabled && !isSearchMode,
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => {
      // Return the cursor for the next page, or undefined if no more pages
      return lastPage.has_more ? lastPage.cursor : undefined;
    },
    staleTime: 0, // Always refetch on window focus
    refetchOnWindowFocus: true, // Refetch when coming back to the page
  });

  // Infinite query for search results (global, not server-specific)
  const searchQuery_infinite = useInfiniteQuery({
    queryKey: ["infinite-search", searchQuery],
    queryFn: ({ pageParam }) =>
      apiClient.searchKnoks(searchQuery!, pageParam),
    enabled: false, // Manual control with debouncing
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => {
      return lastPage.has_more ? lastPage.cursor : undefined;
    },
    staleTime: 1000 * 60 * 5, // Cache for 5 minutes
  });

  // Debounced search effect
  useEffect(() => {
    if (!searchQuery || !enabled) {
      setIsTyping(false);
      return;
    }

    setIsTyping(true);

    const timeoutId = setTimeout(() => {
      searchQuery_infinite.refetch();
      setTimeout(() => setIsTyping(false), 50);
    }, 300); // Debounce by 300ms

    return () => clearTimeout(timeoutId);
  }, [searchQuery, enabled, searchQuery_infinite]);

  // Choose the appropriate query based on mode
  const activeQuery = isSearchMode ? searchQuery_infinite : timelineQuery;

  // Flatten all pages into a single array of knoks
  const knoks: KnokDto[] = activeQuery.data?.pages.flatMap(page => page.knoks) ?? [];

  // Determine loading state
  const isLoading = isSearchMode 
    ? (searchQuery_infinite.isLoading || isTyping)
    : timelineQuery.isLoading;

  return {
    knoks,
    isLoading,
    isFetchingNextPage: activeQuery.isFetchingNextPage,
    hasNextPage: activeQuery.hasNextPage ?? false,
    error: activeQuery.error,
    fetchNextPage: activeQuery.fetchNextPage,
    refetch: () => {
      if (isSearchMode) {
        searchQuery_infinite.refetch();
      } else {
        timelineQuery.refetch();
      }
    },
  };
}