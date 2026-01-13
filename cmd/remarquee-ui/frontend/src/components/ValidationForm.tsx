import { useAppSelector, useAppDispatch } from '../store/hooks';
import { updateReviewStatus, updateReviewNotes, submitValidation, resetReview } from '../store/validationSlice';

function ValidationForm() {
  const dispatch = useAppDispatch();
  const { selectedDocumentId } = useAppSelector(state => state.documents);
  const { jobs } = useAppSelector(state => state.render);
  const { currentReview, loading } = useAppSelector(state => state.validation);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!selectedDocumentId) {
      alert('Please select a document first');
      return;
    }

    const actions = jobs.map(job => `${job.action}: ${job.outputPath}`);
    
    dispatch(submitValidation({
      documentId: selectedDocumentId,
      actions,
      review: currentReview,
      timestamp: Date.now(),
    }));
  };

  const handleReset = () => {
    dispatch(resetReview());
  };

  return (
    <div className="validation-form">
      <h2>Validation</h2>
      
      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label>Status:</label>
          <div className="radio-group">
            <label>
              <input
                type="radio"
                name="status"
                value="pass"
                checked={currentReview.status === 'pass'}
                onChange={() => dispatch(updateReviewStatus('pass'))}
              />
              PASS
            </label>
            
            <label>
              <input
                type="radio"
                name="status"
                value="fail"
                checked={currentReview.status === 'fail'}
                onChange={() => dispatch(updateReviewStatus('fail'))}
              />
              FAIL
            </label>
            
            <label>
              <input
                type="radio"
                name="status"
                value="unknown"
                checked={currentReview.status === 'unknown'}
                onChange={() => dispatch(updateReviewStatus('unknown'))}
              />
              UNKNOWN
            </label>
          </div>
        </div>

        <div className="form-group">
          <label htmlFor="notes">Notes:</label>
          <textarea
            id="notes"
            rows={6}
            value={currentReview.notes}
            onChange={(e) => dispatch(updateReviewNotes(e.target.value))}
            placeholder="Describe what you validated and any issues found..."
          />
        </div>

        <div className="form-actions">
          <button 
            type="submit" 
            className="btn-primary"
            disabled={!selectedDocumentId || loading}
          >
            {loading ? 'Saving...' : 'Save Validation'}
          </button>
          
          <button 
            type="button" 
            className="btn-secondary"
            onClick={handleReset}
          >
            Reset
          </button>
        </div>
      </form>
    </div>
  );
}

export default ValidationForm;

