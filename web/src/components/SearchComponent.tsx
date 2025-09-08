import React, { useState, useEffect, useRef } from "react";
import { Search, X } from "lucide-react";
import type { KnokDto } from "../api/types";
import { QueryClient, useQuery } from "@tanstack/react-query";
import { apiClient } from "@/api/client";
export function SearchComponent() {
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  // const [suggestions, setSuggestions] = useState<KnokDto[]>([]);
  // const [isTestLoading, setIsLoading] = useState(false);
  const inputRef = useRef(null);

  //   Why useQuery with enabled: false:

  //   - Manual control: Only runs when you call refetch()
  //   - Caching: Identical searches return cached results
  //   instantly
  //   - Deduplication: Multiple identical searches trigger
  //   only one request
  //   - Background refetching: Can update stale results
  //   automatically
  //   - Proper semantics: GET requests should use useQuery

  //   Current Issues in Your Component:

  //   1. Double implementation: You have both useQuery AND a
  //    manual searchKnoks function
  //   2. useQuery runs immediately: Your current useQuery
  //   will fire on every searchQuery change
  //   3. Mock data conflict: You're showing mock data
  //   instead of real API results

  // ╭──────────────────────────────────────────────────────────╮
  // │ Ready to code?                                           │
  // │                                                          │
  // │ Here is Claude's plan:                                   │
  // │ ╭──────────────────────────────────────────────────────╮ │
  // │ │ Search Component Refactoring Plan                    │ │
  // │ │                                                      │ │
  // │ │ 1. Fix the useQuery Implementation                   │ │
  // │ │                                                      │ │
  // │ │ - Remove the unused useQuery on lines 13-20 (it      │ │
  // │ │ conflicts with manual search)                        │ │
  // │ │ - OR replace the manual search with a properly       │ │
  // │ │ configured useQuery                                  │ │
  // │ │                                                      │ │
  // │ │ 2. Recommended Approach: useQuery with enabled:      │ │
  // │ │ false                                                │ │
  // │ │                                                      │ │
  // │ │ Replace current implementation with:                 │ │
  // │ │ const { data, isLoading, refetch } = useQuery({      │ │
  // │ │   queryKey: ["search-knoks", searchQuery],           │ │
  // │ │   queryFn: () => apiClient.searchKnoks(searchQuery), │ │
  // │ │   enabled: false, // Don't run automatically         │ │
  // │ │   staleTime: 1000 * 60 * 5 // Cache results for 5    │ │
  // │ │ minutes                                              │ │
  // │ │ });                                                  │ │
  // │ │                                                      │ │
  // │ │ 3. Update Search Logic                               │ │
  // │ │                                                      │ │
  // │ │ - Remove searchKnoks function and suggestions state  │ │
  // │ │ - Replace searchKnoks(searchQuery) call with         │ │
  // │ │ refetch() in the debounced effect                    │ │
  // │ │ - Use data?.knoks || [] instead of suggestions       │ │
  // │ │ - Remove setIsLoading calls (use React Query's       │ │
  // │ │ isLoading)                                           │ │
  // │ │                                                      │ │
  // │ │ 4. Benefits of This Approach                         │ │
  // │ │                                                      │ │
  // │ │ - Proper caching: Search results cached for 5        │ │
  // │ │ minutes                                              │ │
  // │ │ - Deduplication: Identical searches don't hit the    │ │
  // │ │ server twice                                         │ │
  // │ │ - Loading states: React Query manages loading state  │ │
  // │ │ automatically                                        │ │
  // │ │ - Error handling: Built-in error states              │ │
  // │ │ - Background updates: Can refresh stale data         │ │
  // │ │ automatically                                        │ │
  // │ │                                                      │ │
  // │ │ 5. Alternative: Keep Manual Approach                 │ │
  // │ │                                                      │ │
  // │ │ If you prefer manual control:                        │ │
  // │ │ - Remove the conflicting useQuery entirely           │ │
  // │ │ - Keep the current searchKnoks function              │ │
  // │ │ - Replace mock data with                             │ │
  // │ │ apiClient.searchKnoks(searchTerm) call

  // │ │ const { data, isLoading, refetch } = useQuery({      │ │
  // │ │   queryKey: ["search-knoks", searchQuery],           │ │
  // │ │   queryFn: () => apiClient.searchKnoks(searchQuery), │ │
  // │ │   enabled: false, // Don't run automatically         │ │
  // │ │   staleTime: 1000 * 60 * 5 // Cache results for 5    │ │
  // │ │ minutes                                              │ │
  // │ │ });
  const {
    data: { knoks = [] } = {},
    isLoading,
    error,
    isError,
    refetch,
  } = useQuery({
    queryKey: ["search-knoks", searchQuery],
    queryFn: () => apiClient.searchKnoks(searchQuery),
    enabled: false,
    staleTime: 1000 * 60 * 5,
  });

  useEffect(() => {
    const timeoutId = setTimeout(() => {
      if (searchQuery) {
        refetch();
      }
    }, 300); // Debounce input by 300ms

    return () => clearTimeout(timeoutId);
  }, [searchQuery]);

  useEffect(() => {
    if (isSearchOpen && inputRef.current) {
      (inputRef.current as HTMLInputElement).focus();
    }
  }, [isSearchOpen]);

  useEffect(() => {
    const handleEscape = (e: { key: string }) => {
      if (e.key === "Escape") {
        closeSearch();
      }
    };

    if (isSearchOpen) {
      document.addEventListener("keydown", handleEscape);
      return () => document.removeEventListener("keydown", handleEscape);
    }
  }, [isSearchOpen]);

  const openSearch = () => {
    setIsSearchOpen(true);
  };

  const closeSearch = () => {
    setIsSearchOpen(false);
    setSearchQuery("");
  };

  const handleSuggestionClick = (suggestion: string) => {
    setSearchQuery(suggestion);
    console.log("Selected:", suggestion);
    closeSearch();
  };

  return (
    <>
      <button
        onClick={openSearch}
        className="flex items-center justify-center p-2 rounded-lg bg-neutral-700 text-neutral-400 hover:bg-neutral-600 hover:text-white transition-colors"
        aria-label="Search knoks"
      >
        <Search className="h-5 w-5" />
      </button>

      {isSearchOpen && (
        <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm">
          <div className="absolute top-4 left-4 right-4 bg-stone-900 rounded-lg border-b border-neutral-700">
            <div className="flex items-center gap-4 p-4">
              {/* Searchbar input container with relative to place icon */}
              <div className="flex-1 relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-neutral-400 h-5 w-5" />
                <input
                  ref={inputRef}
                  type="text"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="Knok Knok.."
                  className="w-full pl-12 pr-12 py-3 rounded-lg bg-neutral-700 text-white placeholder-neutral-400 focus:outline-none focus:ring-2 focus:ring-knok-accent transition-color text-lg"
                />
                <button
                  onClick={closeSearch}
                  className="absolute right-3 top-1/2 transform -translate-y-1/2 rounded-lg text-neutral-400 hover:text-white hover:bg-neutral-700 transition-colors"
                >
                  <X className="h-6 w-6" />
                </button>
              </div>
            </div>
            {/* Search results container with a max-height of 96 and overflow-y scrollbar if it overflows*/}
            <div className="max-h-96 overflow-y auto">
              {/* // loader */}
              {isLoading && (
                <div className="px-4 py-8 text-center text-neutral-400">
                  <div className="inline-block animate-spin rounded-full h-6 w-6 border-b-2 border-knok-accent"></div>
                  <p className="mt-2">Knockin...</p>
                </div>
              )}
              {!isLoading && knoks.length > 0 && (
                // Search result container that is a div containing a button that also contains a span
                <div className="py-2">
                  {knoks.map((suggestion, index) => (
                    <button
                      key={index}
                      onClick={() => handleSuggestionClick(suggestion.url)}
                      className="w-full px-4 py-3 text-left text-white hover:bg-neutral-700 transition-colors flex items-center gap-3"
                    >
                      <Search className="h-4 w-4 text-neutral-400 flex-shrink-0" />
                      <span>{suggestion.title}</span>
                    </button>
                  ))}
                </div>
              )}
              {/* No results display */}
              {!isLoading && searchQuery && knoks.length === 0 && (
                <div className="px-4 py-8 text-center text-neutral-400">
                  <p>No results found for "{searchQuery}"</p>
                </div>
              )}
              {/* Placeholder no search yet */}
              {!searchQuery && !isLoading && (
                <div className="px-4 py-8 text-center text-neutral-400">
                  <p>Start typing to search knoks.. TODO Knok logo</p>
                </div>
              )}
              {isError && (
                <div className="px-4 py-8 text-center text-red-400">
                  <p>Search failed. Please try again.</p>
                  {error?.message && (
                    <p className="text-sm mt-1">{error.message}</p>
                  )}
                </div>
              )}
            </div>
          </div>
          {/* Click outside to close */}
          <div className="absolute inset-0 -z-10" onClick={closeSearch} />
        </div>
      )}
    </>
  );
}
