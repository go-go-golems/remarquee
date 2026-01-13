import { useAppSelector } from '../store/hooks';

function ValidationGuidance() {
  const { inspectResult } = useAppSelector(state => state.documents);

  return (
    <div className="guidance-panel">
      <h2>Validation Checklist</h2>
      
      <h3>Document Info</h3>
      {inspectResult ? (
        <div style={{ fontSize: '0.9rem', marginBottom: '1rem' }}>
          <p><strong>Schema:</strong> {inspectResult.schema}</p>
          <p><strong>Type:</strong> {inspectResult.docType}</p>
          <p><strong>Pages:</strong> {inspectResult.pageCount}</p>
        </div>
      ) : (
        <p style={{ fontSize: '0.9rem', color: '#7f8c8d' }}>Select a document first</p>
      )}

      <h3>What to Check</h3>
      
      <div className="checklist-item">
        <span>UI page order matches expected order</span>
      </div>
      
      <div className="checklist-item">
        <span>Duplicated pages appear multiple times</span>
      </div>
      
      <div className="checklist-item">
        <span>Inserted pages show as blank templates</span>
      </div>
      
      <div className="checklist-item">
        <span>Source PDF page mapping is correct</span>
      </div>
      
      {inspectResult?.docType === 'PDF' && (
        <>
          <div className="checklist-item">
            <span>Background PDF has all pages</span>
          </div>
          
          <div className="checklist-item">
            <span>Page dimensions are correct</span>
          </div>
        </>
      )}
      
      {inspectResult?.schema === 'legacy' && (
        <>
          <h3>Legacy (V3/V5)</h3>
          <div className="checklist-item">
            <span>Annotations render correctly</span>
          </div>
          <div className="checklist-item">
            <span>Stroke colors match original</span>
          </div>
        </>
      )}
      
      {inspectResult?.schema === 'cPages' && (
        <>
          <h3>Modern (cPages/V6)</h3>
          <div className="checklist-item">
            <span>Redirection map applied correctly</span>
          </div>
          <div className="checklist-item">
            <span>Deleted pages are skipped</span>
          </div>
        </>
      )}

      <h3>Common Issues</h3>
      <ul style={{ fontSize: '0.85rem', color: '#555' }}>
        <li>Missing pages → check pageCount vs actual pages</li>
        <li>Wrong order → verify redirection/cPages logic</li>
        <li>Blank when expected content → check .rm file exists</li>
        <li>Coordinate misalignment → verify transform constants</li>
      </ul>
    </div>
  );
}

export default ValidationGuidance;

