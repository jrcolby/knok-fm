import { Route, BrowserRouter as Router, Routes } from "react-router";
import { KnokTimeline } from "./pages/KnokTimeline";

function App() {
  return (
    <Router>
      <div className="min-h-screen bg-gray-100">
        <header className="bg-white shadow">
          <div className="max-w-7xl mx-auto py-6 px-4">
            <h1 className="text-3xl font-bold text-gray-900">Knok FM</h1>
          </div>
        </header>
        <main>
          <Routes>
            <Route path="/" element={<KnokTimeline />} />
            {/* <Route
              path="/servers"
              element={<div className="p-8">Server List (Coming Soon)</div>}
            /> */}
            <Route path="/knoks" element={<KnokTimeline />} />
          </Routes>
        </main>
      </div>
    </Router>
  );
}

export default App;
