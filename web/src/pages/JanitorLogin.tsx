import { useState } from 'react';
import type { FormEvent } from 'react';
import { useNavigate } from 'react-router';
import { useAdmin } from '../contexts/AdminContext';

export function JanitorLogin() {
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const { login } = useAdmin();
  const navigate = useNavigate();

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    setError('');

    if (!password.trim()) {
      setError('Password is required');
      return;
    }

    // Store the API key (which is the password)
    login(password);

    // Redirect to main knoks page in admin mode
    navigate('/knoks');
  };

  return (
    <div className="min-h-screen bg-stone-950 flex items-center justify-center p-4">
      <div className="bg-stone-900 rounded-lg shadow-xl max-w-md w-full p-8">
        <h1 className="text-2xl font-bold text-white mb-6 text-center">
          Janitor Access
        </h1>

        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label
              htmlFor="password"
              className="block text-sm font-medium text-stone-300 mb-2"
            >
              Admin Password
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-4 py-2 rounded bg-stone-800 text-white border border-stone-700 focus:border-blue-500 focus:outline-none"
              placeholder="Enter admin password"
              autoFocus
            />
          </div>

          {error && (
            <p className="text-red-500 text-sm mb-4">{error}</p>
          )}

          <button
            type="submit"
            className="w-full bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded transition-colors"
          >
            Login
          </button>
        </form>

        <p className="mt-6 text-sm text-stone-400 text-center">
          Enter the admin API key to access janitor controls
        </p>
      </div>
    </div>
  );
}
