import { Route, BrowserRouter as Router, Routes } from "react-router";
import { KnokTimeline } from "./pages/KnokTimeline";
import { Header } from "./components/Header";
import { AdminProvider } from "./contexts/AdminContext";
import { JanitorLogin } from "./pages/JanitorLogin";

function App() {
  return (
    <AdminProvider>
      <Router>
        <div className="min-h-screen bg-neutral-800 text-white">
          <Header />
          <main>
            <Routes>
              <Route path="/" element={<KnokTimeline />} />
              {/* <Route
                path="/servers"
                element={<div className="p-8">Server List (Coming Soon)</div>}
              /> */}
              <Route path="/knoks" element={<KnokTimeline />} />
              <Route path="/janitor" element={<JanitorLogin />} />
            </Routes>
          </main>
        </div>
      </Router>
    </AdminProvider>
  );
}

export default App;
