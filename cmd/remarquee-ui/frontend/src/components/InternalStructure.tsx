import { useState, useEffect } from 'react';
import { useAppSelector } from '../store/hooks';

interface RMFileInfo {
  pageId: string;
  filename: string;
  size: number;
  version?: string;
}

interface InternalStructure {
  contentJson: string;
  metadataJson: string;
  rmFiles: RMFileInfo[];
  allFiles: string[];
}

function InternalStructure() {
  const { selectedDocumentId } = useAppSelector(state => state.documents);
  const [structure, setStructure] = useState<InternalStructure | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<{ [key: string]: boolean }>({});

  useEffect(() => {
    if (selectedDocumentId) {
      fetchStructure(selectedDocumentId);
    } else {
      setStructure(null);
    }
  }, [selectedDocumentId]);

  const fetchStructure = async (docId: string) => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(`/api/document/${docId}/structure`);
      if (!response.ok) {
        throw new Error('Failed to fetch internal structure');
      }
      const data = await response.json();
      setStructure(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  const toggleSection = (section: string) => {
    setExpanded(prev => ({ ...prev, [section]: !prev[section] }));
  };

  if (!selectedDocumentId) {
    return null;
  }

  if (loading) {
    return (
      <div className="internal-structure">
        <h3>Internal Structure</h3>
        <p style={{ color: '#7f8c8d', fontSize: '0.9rem' }}>Loading...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="internal-structure">
        <h3>Internal Structure</h3>
        <p style={{ color: '#e74c3c', fontSize: '0.9rem' }}>Error: {error}</p>
      </div>
    );
  }

  if (!structure) {
    return null;
  }

  return (
    <div className="internal-structure">
      <h3>Internal Structure</h3>

      {/* .rm Files */}
      <details className="structure-section" open={expanded['rmFiles']}>
        <summary onClick={() => toggleSection('rmFiles')}>
          <strong>Annotation Files ({structure.rmFiles.length})</strong>
        </summary>
        <div className="structure-content">
          {structure.rmFiles.length === 0 ? (
            <p style={{ fontSize: '0.85rem', color: '#7f8c8d' }}>No .rm files found</p>
          ) : (
            <table className="rm-files-table">
              <thead>
                <tr>
                  <th>Page ID</th>
                  <th>Version</th>
                  <th>Size</th>
                </tr>
              </thead>
              <tbody>
                {structure.rmFiles.map(rm => (
                  <tr key={rm.filename}>
                    <td><code style={{ fontSize: '0.85rem' }}>{rm.pageId.substring(0, 8)}...</code></td>
                    <td>
                      <span style={{ 
                        background: rm.version === 'V6' ? '#27ae60' : rm.version === 'V3' || rm.version === 'V5' ? '#f39c12' : '#95a5a6',
                        color: 'white',
                        padding: '0.15rem 0.4rem',
                        borderRadius: '8px',
                        fontSize: '0.75rem',
                        fontWeight: 'bold'
                      }}>
                        {rm.version || '?'}
                      </span>
                    </td>
                    <td style={{ fontSize: '0.85rem' }}>{(rm.size / 1024).toFixed(1)} KB</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </details>

      {/* .content JSON */}
      <details className="structure-section">
        <summary>
          <strong>.content JSON</strong>
        </summary>
        <div className="structure-content">
          <pre className="json-viewer">{structure.contentJson}</pre>
        </div>
      </details>

      {/* .metadata JSON */}
      <details className="structure-section">
        <summary>
          <strong>.metadata JSON</strong>
        </summary>
        <div className="structure-content">
          <pre className="json-viewer">{structure.metadataJson}</pre>
        </div>
      </details>

      {/* All Files */}
      <details className="structure-section">
        <summary>
          <strong>All Files ({structure.allFiles.length})</strong>
        </summary>
        <div className="structure-content">
          <ul style={{ fontSize: '0.85rem', maxHeight: '300px', overflowY: 'auto' }}>
            {structure.allFiles.map(file => (
              <li key={file} style={{ fontFamily: 'monospace' }}>{file}</li>
            ))}
          </ul>
        </div>
      </details>
    </div>
  );
}

export default InternalStructure;

