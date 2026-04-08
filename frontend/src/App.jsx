import { Routes, Route, Navigate } from 'react-router-dom'
import Sidebar from './components/Sidebar'
import { useAuth } from './components/AuthProvider'
import Dashboard from './pages/Dashboard'
import Companies from './pages/Companies'
import CompanyDetail from './pages/CompanyDetail'
import NewsFeed from './pages/NewsFeed'
import { AnalysisList, AnalysisDetail } from './pages/Analysis'

function AdminRoute({ children }) {
  const profile = useAuth()
  if (!profile?.is_admin) {
    return <Navigate to="/news" replace />
  }
  return children
}

export default function App() {
  const profile = useAuth()
  const isAdmin = profile?.is_admin

  return (
    <div className="app">
      <Sidebar />
      <main className="main-content">
        <Routes>
          <Route path="/" element={isAdmin ? <Dashboard /> : <Navigate to="/news" replace />} />
          <Route path="/companies" element={<AdminRoute><Companies /></AdminRoute>} />
          <Route path="/company/:ticker" element={<AdminRoute><CompanyDetail /></AdminRoute>} />
          <Route path="/news" element={<NewsFeed />} />
          <Route path="/analysis" element={<AnalysisList />} />
          <Route path="/analysis/:id" element={<AnalysisDetail />} />
        </Routes>
      </main>
    </div>
  )
}
