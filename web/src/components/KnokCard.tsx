import type { KnokDto } from "../api/types";

interface KnokCardProps {
  knok: KnokDto;
}

export function KnokCard({ knok }: KnokCardProps) {
  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  const getDomainFromUrl = (url: string) => {
    try {
      return new URL(url).hostname.replace("www.", "");
    } catch {
      return "Unknown";
    }
  };

  return (
    <div className="bg-white rounded-lg shadow-md p-6 hover:shadow-lg transition-shadow">
      <div className="flex items-start justify-between mb-3">
        <h3 className="text-lg font-semibold text-gray-900 line-clamp-2">
          {knok.title}
        </h3>
        <span className="text-sm text-gray-500 ml-4 whitespace-nowrap">
          {getDomainFromUrl(knok.url)}
        </span>
      </div>

      <div className="text-sm text-gray-600 mb-4">
        Posted: {formatDate(knok.posted_at)}
      </div>

      <a
        href={knok.url}
        target="_blank"
        rel="noopener noreferrer"
        className="inline-flex items-center px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700 transition-colors"
      >
        Listen
        <svg
          className="ml-2 h-4 w-4"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M10 6H6a2 2 0 00-2 2v6a2 2 0 002 2h10a2 2 0 002-2v-6a2 2 0 00-2-2h-4M10 6V4a2 2 0 114 0v2M10 6h4"
          />
        </svg>
      </a>
    </div>
  );
}
