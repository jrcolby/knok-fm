import { useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { apiClient } from "../api/client";

export function useKnokData(serverId: string, searchQuery: string, randomTrigger: string | null) {
  const [isTyping, setIsTyping] = useState(false);
  
  const isSearchMode = !!searchQuery;
  const isRandomMode = !!randomTrigger;

  // Query for normal timeline knoks
  const {
    data: timelineData,
    isLoading: isTimelineLoading,
    error: timelineError,
  } = useQuery({
    queryKey: ["knoks", serverId],
    queryFn: () => apiClient.getKnoksByServer(serverId),
    enabled: !searchQuery && !isRandomMode, // Only fetch when not searching or random
  });

  // Query for search results
  const {
    data: searchData,
    isLoading: isSearchLoading,
    error: searchError,
    refetch: searchRefetch,
  } = useQuery({
    queryKey: ["search-knoks", searchQuery],
    queryFn: () => apiClient.searchKnoks(searchQuery),
    enabled: false, // Manual control
    staleTime: 1000 * 60 * 5, // Cache for 5 minutes
  });

  // Query for random knok
  const {
    data: randomData,
    isLoading: isRandomLoading,
    error: randomError,
    refetch: refetchRandom,
  } = useQuery({
    queryKey: ["random-knok"],
    queryFn: () => apiClient.getRandomKnok(),
    enabled: false, // Manual control only
    staleTime: 0, // Don't cache random results
  });

  // Debounced search effect with typing state management
  useEffect(() => {
    if (!searchQuery) {
      setIsTyping(false);
      return;
    }

    setIsTyping(true);

    const timeoutId = setTimeout(() => {
      searchRefetch();
      // Keep showing loading for a bit after search completes
      setTimeout(() => setIsTyping(false), 50);
    }, 300); // Debounce by 300ms

    return () => clearTimeout(timeoutId);
  }, [searchQuery, searchRefetch]);

  // Effect to trigger random fetch when random trigger changes
  useEffect(() => {
    if (randomTrigger) {
      refetchRandom();
    }
  }, [randomTrigger, refetchRandom]);

  // Determine which data to use
  const knoksData = isRandomMode
    ? randomData
    : isSearchMode
    ? searchData
    : timelineData;
    
  const isLoading = isRandomMode
    ? isRandomLoading
    : isSearchMode
    ? isSearchLoading || isTyping
    : isTimelineLoading;
    
  const error = isRandomMode
    ? randomError
    : isSearchMode
    ? searchError
    : timelineError;

  return {
    knoksData,
    isLoading,
    error,
    isSearchMode,
    isRandomMode,
  };
}