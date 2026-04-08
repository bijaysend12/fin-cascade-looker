import { Routes, Route } from 'react-router-dom'
import Sidebar from './components/Sidebar'
import Dashboard from './pages/Dashboard'
import Companies from './pages/Companies'
import CompanyDetail from './pages/CompanyDetail'
import NewsFeed from './pages/NewsFeed'
import { AnalysisList, AnalysisDetail } from './pages/Analysis'

export default function App() {
  return (
    <div className="app">
      <Sidebar />
      <main className="main-content">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/companies" element={<Companies />} />
          <Route path="/company/:ticker" element={<CompanyDetail />} />
          <Route path="/news" element={<NewsFeed />} />
          <Route path="/analysis" element={<AnalysisList />} />
          <Route path="/analysis/:id" element={<AnalysisDetail />} />
        </Routes>
      </main>
    </div>
  )
}
