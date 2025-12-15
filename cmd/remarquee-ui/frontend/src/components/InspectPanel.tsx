import { useAppSelector } from '../store/hooks';

function InspectPanel() {
  const { inspectResult, loading } = useAppSelector(state => state.documents);

  if (!inspectResult && !loading) {
    return <div className="inspect-panel empty">Select a document to inspect</div>;
  }

  if (loading) {
    return <div className="inspect-panel loading">Inspecting...</div>;
  }

  return (
    <div className="inspect-panel">
      <h2>Document Inspection</h2>
      
      <div className="inspect-metadata">
        <p><strong>UUID:</strong> {inspectResult!.uuid}</p>
        <p><strong>Schema:</strong> {inspectResult!.schema}</p>
        <p><strong>Doc Type:</strong> {inspectResult!.docType}</p>
        <p><strong>Page Count:</strong> {inspectResult!.pageCount}</p>
        <p><strong>Has Payload PDF:</strong> {inspectResult!.hasPayloadPDF ? 'Yes' : 'No'}</p>
      </div>

      <h3>Pages</h3>
      <table className="pages-table">
        <thead>
          <tr>
            <th>UI Index</th>
            <th>Page ID</th>
            <th>Source PDF Page</th>
            <th>Template</th>
          </tr>
        </thead>
        <tbody>
          {inspectResult!.pages.map(page => (
            <tr key={page.pageId}>
              <td>{page.index}</td>
              <td><code>{page.pageId.substring(0, 8)}...</code></td>
              <td>{page.sourcePdfPage === -1 ? '(inserted)' : page.sourcePdfPage}</td>
              <td>{page.template || '—'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default InspectPanel;

