import { useEffect } from 'react'
import './App.css'
import { useAppDispatch } from './store/hooks'
import { fetchTestDocuments } from './store/documentsSlice'
import DocumentSelector from './components/DocumentSelector'
import InspectPanel from './components/InspectPanel'
import RenderActions from './components/RenderActions'
import PDFViewer from './components/PDFViewer'
import ValidationForm from './components/ValidationForm'
import ValidationGuidance from './components/ValidationGuidance'

function App() {
  const dispatch = useAppDispatch();

  useEffect(() => {
    // Fetch test documents on mount
    dispatch(fetchTestDocuments());
  }, [dispatch]);

  return (
    <div className="app-container">
      <header className="app-header">
        <h1>remarquee-ui — rmdoc validation tool</h1>
      </header>
      
      <div className="app-layout">
        <aside className="sidebar">
          <DocumentSelector />
          <RenderActions />
        </aside>
        
        <main className="main-panel">
          <InspectPanel />
          <PDFViewer />
          <ValidationForm />
        </main>

        <aside className="guidance-panel">
          <ValidationGuidance />
        </aside>
      </div>
    </div>
  )
}

export default App
