import { Routes, Route, Navigate } from 'react-router-dom'
import Sidebar from './components/Sidebar'
import { useAuth } from './components/AuthProvider'
import HomePage from './pages/HomePage'
import StocksPage from './pages/StocksPage'
import Dashboard from './pages/Dashboard'
import Companies from './pages/Companies'
import CompanyDetail from './pages/CompanyDetail'
import NewsFeed from './pages/NewsFeed'
import { AnalysisList, AnalysisDetail } from './pages/Analysis'

function AdminRoute({ children }) {
  const profile = useAuth()
  if (!profile?.is_admin) {
    return <Navigate to="/" replace />
  }
  return children
}

export default function App() {
  return (
    <div className="app">
      <Sidebar />
      <main className="main-content">
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/stocks" element={<StocksPage />} />
          <Route path="/news" element={<NewsFeed />} />
          <Route path="/analysis" element={<AnalysisList />} />
          <Route path="/analysis/:id" element={<AnalysisDetail />} />
          <Route path="/admin" element={<AdminRoute><Dashboard /></AdminRoute>} />
          <Route path="/companies" element={<AdminRoute><Companies /></AdminRoute>} />
          <Route path="/company/:ticker" element={<AdminRoute><CompanyDetail /></AdminRoute>} />
        </Routes>
      </main>
    </div>
  )
}
