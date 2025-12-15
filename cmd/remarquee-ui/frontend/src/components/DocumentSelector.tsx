import { useAppSelector, useAppDispatch } from '../store/hooks';
import { selectDocument, fetchInspect } from '../store/documentsSlice';

function DocumentSelector() {
  const dispatch = useAppDispatch();
  const { testDocuments, selectedDocumentId, loading } = useAppSelector(state => state.documents);

  const handleSelectDocument = (docId: string) => {
    dispatch(selectDocument(docId));
    dispatch(fetchInspect(docId));
  };

  if (loading && testDocuments.length === 0) {
    return <div className="document-selector loading">Loading test documents...</div>;
  }

  return (
    <div className="document-selector">
      <h2>Test Documents</h2>
      <ul className="document-list">
        {testDocuments.map(doc => (
          <li 
            key={doc.id}
            className={selectedDocumentId === doc.id ? 'selected' : ''}
          >
            <button 
              onClick={() => handleSelectDocument(doc.id)}
              disabled={loading}
            >
              <strong>{doc.name}</strong>
              <small>{doc.schema} / {doc.docType} / {doc.expectedPages}p</small>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}

export default DocumentSelector;

