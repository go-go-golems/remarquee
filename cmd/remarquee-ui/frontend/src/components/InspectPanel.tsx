import { useAppSelector } from '../store/hooks';

function InspectPanel() {
  const { inspectResult, loading, selectedDocumentId, testDocuments } = useAppSelector(state => state.documents);

  if (!inspectResult && !loading) {
    return <div className="inspect-panel empty">Select a document to inspect</div>;
  }

  if (loading) {
    return <div className="inspect-panel loading">Inspecting...</div>;
  }

  const selectedDoc = testDocuments.find(d => d.id === selectedDocumentId);

  return (
    <div className="inspect-panel">
      <h2>Document Inspection</h2>
      
      {selectedDoc && (
        <div className="doc-description" style={{ 
          background: '#f0f8ff', 
          padding: '1rem', 
          borderRadius: '4px', 
          marginBottom: '1.5rem',
          borderLeft: '3px solid #3498db'
        }}>
          <h3 style={{ marginTop: 0, fontSize: '1rem', color: '#2c3e50' }}>
            {selectedDoc.name}
          </h3>
          <p style={{ margin: '0.5rem 0', fontSize: '0.9rem', color: '#555' }}>
            {selectedDoc.description}
          </p>
          <p style={{ margin: '0.5rem 0', fontSize: '0.85rem', color: '#7f8c8d' }}>
            <strong>Expected pages:</strong> {selectedDoc.expectedPages}
          </p>
        </div>
      )}
      
      <div className="inspect-metadata" style={{ 
        display: 'grid', 
        gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', 
        gap: '1rem',
        marginBottom: '1.5rem'
      }}>
        <div>
          <p style={{ margin: '0.25rem 0' }}><strong>UUID:</strong></p>
          <p style={{ margin: '0.25rem 0', fontSize: '0.9rem', color: '#555' }}>
            <code style={{ background: '#ecf0f1', padding: '0.2rem 0.4rem', borderRadius: '3px' }}>
              {inspectResult!.uuid}
            </code>
          </p>
        </div>
        <div>
          <p style={{ margin: '0.25rem 0' }}><strong>Schema:</strong></p>
          <p style={{ margin: '0.25rem 0', fontSize: '0.9rem' }}>
            <span style={{ 
              background: inspectResult!.schema === 'cPages' ? '#27ae60' : '#f39c12',
              color: 'white',
              padding: '0.2rem 0.6rem',
              borderRadius: '12px',
              fontSize: '0.85rem'
            }}>
              {inspectResult!.schema}
            </span>
          </p>
        </div>
        <div>
          <p style={{ margin: '0.25rem 0' }}><strong>Doc Type:</strong></p>
          <p style={{ margin: '0.25rem 0', fontSize: '0.9rem' }}>
            <span style={{ 
              background: inspectResult!.docType === 'PDF' ? '#3498db' : '#9b59b6',
              color: 'white',
              padding: '0.2rem 0.6rem',
              borderRadius: '12px',
              fontSize: '0.85rem'
            }}>
              {inspectResult!.docType}
            </span>
          </p>
        </div>
        <div>
          <p style={{ margin: '0.25rem 0' }}><strong>Page Count:</strong></p>
          <p style={{ margin: '0.25rem 0', fontSize: '1.2rem', fontWeight: 'bold', color: '#2c3e50' }}>
            {inspectResult!.pageCount}
          </p>
        </div>
        <div>
          <p style={{ margin: '0.25rem 0' }}><strong>Has Payload PDF:</strong></p>
          <p style={{ margin: '0.25rem 0', fontSize: '0.9rem' }}>
            {inspectResult!.hasPayloadPDF ? '✓ Yes' : '✗ No'}
          </p>
        </div>
      </div>

      <h3>Page Details</h3>
      <div style={{ overflowX: 'auto' }}>
        <table className="pages-table">
          <thead>
            <tr>
              <th>UI Index</th>
              <th>Page ID</th>
              <th>Source PDF Page</th>
              <th>Template</th>
              <th>Type</th>
            </tr>
          </thead>
          <tbody>
            {inspectResult!.pages.map(page => {
              const isDuplicate = inspectResult!.pages.filter(
                p => p.sourcePdfPage === page.sourcePdfPage && page.sourcePdfPage !== -1
              ).length > 1;
              const isInserted = page.sourcePdfPage === -1;
              
              return (
                <tr key={page.pageId} style={{ 
                  background: isInserted ? '#fff3cd' : isDuplicate ? '#d1ecf1' : 'transparent'
                }}>
                  <td>{page.index}</td>
                  <td>
                    <code style={{ fontSize: '0.85rem' }}>
                      {page.pageId.substring(0, 8)}...
                    </code>
                  </td>
                  <td style={{ fontWeight: isInserted || isDuplicate ? 'bold' : 'normal' }}>
                    {page.sourcePdfPage === -1 ? '(inserted)' : page.sourcePdfPage}
                  </td>
                  <td>{page.template || '—'}</td>
                  <td>
                    {isInserted && <span style={{ color: '#856404', fontSize: '0.85rem' }}>Inserted</span>}
                    {isDuplicate && !isInserted && <span style={{ color: '#0c5460', fontSize: '0.85rem' }}>Duplicate</span>}
                    {!isInserted && !isDuplicate && <span style={{ color: '#7f8c8d', fontSize: '0.85rem' }}>Normal</span>}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}

export default InspectPanel;

