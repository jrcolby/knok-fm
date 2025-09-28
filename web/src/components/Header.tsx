import { Search, X } from "lucide-react";
import { useSearchParams } from "react-router";
import { useState, useRef, useEffect } from "react";
import { KnokFmLogo, KnokStar } from "./icons";

export function Header() {
  const [searchParams, setSearchParams] = useSearchParams();
  const searchQuery = searchParams.get("q") || "";
  const [isSearchOpen, setIsSearchOpen] = useState(!!searchQuery);
  const inputRef = useRef<HTMLInputElement>(null);

  // Open search if there's a query in URL
  useEffect(() => {
    setIsSearchOpen(!!searchQuery);
  }, [searchQuery]);

  // Focus input when search opens
  useEffect(() => {
    if (isSearchOpen && inputRef.current) {
      inputRef.current.focus();
    }
  }, [isSearchOpen]);

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    if (value.trim()) {
      setSearchParams({ q: value });
    } else {
      setSearchParams({});
    }
  };

  const openSearch = () => {
    setIsSearchOpen(true);
  };

  const closeSearch = () => {
    setIsSearchOpen(false);
    setSearchParams({});
  };

  const handleRandomKnok = () => {
    // Close search if open and set random parameter with timestamp
    setIsSearchOpen(false);
    setSearchParams({ random: Date.now().toString() });
  };

  const handleHomeClick = () => {
    // Clear all parameters to return to normal timeline
    setIsSearchOpen(false);
    setSearchParams({});
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") {
      closeSearch();
    }
  };

  return (
    <header className="sticky top-0 z-50 bg-neutral-800/80 backdrop-blur-md shadow-lg">
      <div className="max-w-7xl mx-auto px-4 relative min-h-[3.5rem] flex items-center justify-between w-full">
        <div className="flex justify-center gap-2">
          <button
            onClick={handleHomeClick}
            className={`text-knok-accent hover:text-knok-accent/80 transition-all duration-300 cursor-pointer ${
              isSearchOpen ? "opacity-0" : "opacity-100"
            }`}
            aria-label="Return to timeline"
          >
            <KnokFmLogo className="h-8 md:h-10" />
          </button>
        </div>

        <div className="flex items-center gap-2 ml-auto">
          <button
            onClick={handleRandomKnok}
            className={`flex items-center justify-center p-2 text-knok-accent hover:text-knok-accent/80 transition-opacity duration-300 cursor-pointer ${
              isSearchOpen ? "opacity-0" : "opacity-100"
            }`}
            aria-label="Get random knok"
          >
            <KnokStar className="h-5 w-5" />
          </button>

          <button
            onClick={openSearch}
            className={`flex items-center justify-center p-2 text-knok-accent hover:text-knok-accent/80 transition-colors cursor-pointer ${
              isSearchOpen ? "opacity-0 pointer-events-none" : "opacity-100"
            }`}
            aria-label="Open search"
          >
            <Search className="h-5 w-5" />
          </button>
        </div>
      </div>

      {isSearchOpen && (
        <div
          className={`absolute inset-0 flex items-center justify-center px-4 transition-all duration-500 ease-out ${
            isSearchOpen ? "opacity-100 scale-100" : "opacity-0 scale-95"
          }`}
        >
          <div className="relative w-full max-w-2xl">
            <input
              ref={inputRef}
              type="text"
              value={searchQuery}
              onChange={handleSearchChange}
              onKeyDown={handleKeyDown}
              placeholder="Search knoks..."
              className="w-full pl-4 pr-10 py-3 rounded-lg bg-neutral-700 text-white placeholder-neutral-400 focus:outline-none focus:ring-2 focus:ring-knok-accent transition-colors"
            />
            <button
              onClick={closeSearch}
              className="absolute right-3 top-1/2 transform -translate-y-1/2 p-1 rounded-lg text-neutral-400 hover:text-white hover:bg-neutral-600 transition-colors cursor-pointer"
              aria-label="Close search"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
    </header>
  );
}
