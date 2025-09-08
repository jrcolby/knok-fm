import { SearchComponent } from "./SearchComponent";
// container -> constraint -> layout
// Header is styling and semantic meaning
// out div is content width constraints and spacing
// inn div is internal larout
export function Header() {
  return (
    // Header class, semantic container, fills width by default, then expands to fit content
    // It is a block element
    <header className="bg-neutral-800 border-b border-neutral-800 shadow-lg">
      {/* // content wrapper for max width 7xl for content inside, mx-auto to center
      horizontally // py-6 for vertical padding, px-4 for horizontal padding */}
      <div className="max-w-7xl mx-auto py-6 px-4">
        {/* flex box container for logo, search, vertically center them*/}
        {/* // justify between means they go to either end */}
        <div className="flex items-center justify-between">
          {/* // h1 sibling this will be logo */}
          <h1 className="text-md font-bold text-knok-accent">Knok FM</h1>
          {/* // search overlay component within button */}
          <SearchComponent />
        </div>
      </div>
    </header>
  );
}
