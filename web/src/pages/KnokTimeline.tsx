import { useParams } from "react-router";
import { useQuery } from "@tanstack/react-query";
import { apiClient } from "../api/client";
import { KnokCard } from "../components/KnokCard";

const DEFAULT_SERVER_ID =
  import.meta.env.VITE_SERVER_ID || "1404225099841667072";

export function KnokTimeline() {
  const { serverId } = useParams<{ serverId: string }>();
  const actualServerId = serverId || DEFAULT_SERVER_ID;

  const {
    data: knoksData,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["knoks", actualServerId],
    queryFn: () => apiClient.getKnoksByServer(actualServerId),
  });

  if (isLoading) {
    return (
      <div className="max-w-4xl mx-auto p-8">
        <div className="animate-pulse">
          <div className="h-8 bg-gray-200 rounded mb-6"></div>
          <div className="space-y-4">
            {[...Array(3)].map((_, i) => (
              <div key={i} className="bg-gray-200 h-32 rounded-lg"></div>
            ))}
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="max-w-4xl mx-auto p-8">
        <div className="bg-red-50 border border-red-200 rounded-lg p-6">
          <h2 className="text-lg font-semibold text-red-800 mb-2">
            Error loading knoks
          </h2>
          <p className="text-red-600">
            {error instanceof Error
              ? error.message
              : "An unknown error occurred"}
          </p>
          <button
            onClick={() => window.location.reload()}
            className="mt-4 px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700"
          >
            Try Again
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto p-8">
      <h1 className="text-2xl font-bold text-gray-900 mb-6">
        Server Knoks
        {serverId !== actualServerId && (
          <span className="text-sm font-normal text-gray-500 ml-2">
            (Using default server: {DEFAULT_SERVER_ID})
          </span>
        )}
      </h1>

      {!knoksData?.knoks || knoksData.knoks.length === 0 ? (
        <div className="text-center py-12">
          <div className="text-gray-500 text-lg mb-2">No knoks found</div>
          <div className="text-gray-400">
            Share some music links in Discord to see them here!
          </div>
        </div>
      ) : (
        <div className="space-y-6">
          {knoksData.knoks.map((knok) => (
            <KnokCard key={knok.id} knok={knok} />
          ))}

          {knoksData.has_more && (
            <div className="text-center py-4">
              <div className="text-gray-500">More knoks available...</div>
              <div className="text-sm text-gray-400">
                (Infinite scroll coming soon)
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
