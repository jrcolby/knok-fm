export function SearchBar() {
  return (
    <div className="max-w-xl mx-auto p-8">
      <input
        type="text"
        placeholder="Search knoks..."
        className="w-full p-3 rounded bg-neutral-700 text-white placeholder-neutral-400 focus:outline-none focus:ring-2 focus:ring-yellow-400"
      />
    </div>
  );
}
