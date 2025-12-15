import { useAppSelector } from '../store/hooks';

function PDFViewer() {
  const { jobs } = useAppSelector(state => state.render);

  if (jobs.length === 0) {
    return <div className="pdf-viewer empty">No PDFs generated yet</div>;
  }

  // Show the most recent job
  const latestJob = jobs[jobs.length - 1];

  return (
    <div className="pdf-viewer">
      <h2>PDF Output</h2>
      
      <div className="pdf-info">
        <p><strong>Action:</strong> {latestJob.action}</p>
        <p><strong>Document:</strong> {latestJob.documentId}</p>
        <p><strong>Job ID:</strong> {latestJob.jobId}</p>
      </div>

      <div className="pdf-actions">
        <a 
          href={`/api/outputs/${latestJob.outputPath}`}
          target="_blank"
          rel="noopener noreferrer"
          className="btn-primary"
        >
          Open PDF
        </a>
        
        <a 
          href={`/api/outputs/${latestJob.outputPath}`}
          download
          className="btn-secondary"
        >
          Download PDF
        </a>
      </div>

      <div className="pdf-preview">
        <iframe 
          src={`/api/outputs/${latestJob.outputPath}`}
          title="PDF Preview"
          width="100%"
          height="600px"
        />
      </div>

      {jobs.length > 1 && (
        <details className="job-history">
          <summary>Previous outputs ({jobs.length - 1})</summary>
          <ul>
            {jobs.slice(0, -1).reverse().map(job => (
              <li key={job.jobId}>
                <a href={`/api/outputs/${job.outputPath}`} target="_blank" rel="noopener noreferrer">
                  {job.action} — {job.documentId} — {new Date(job.timestamp).toLocaleTimeString()}
                </a>
              </li>
            ))}
          </ul>
        </details>
      )}
    </div>
  );
}

export default PDFViewer;

