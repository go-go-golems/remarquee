import { useAppSelector, useAppDispatch } from '../store/hooks';
import { renderBackground, renderLegacy } from '../store/renderSlice';

function RenderActions() {
  const dispatch = useAppDispatch();
  const { selectedDocumentId } = useAppSelector(state => state.documents);
  const { loading } = useAppSelector(state => state.render);

  const handleRenderBackground = () => {
    if (selectedDocumentId) {
      dispatch(renderBackground(selectedDocumentId));
    }
  };

  const handleRenderLegacy = () => {
    if (selectedDocumentId) {
      dispatch(renderLegacy(selectedDocumentId));
    }
  };

  const disabled = !selectedDocumentId || loading;

  return (
    <div className="render-actions">
      <h2>Render</h2>
      <div className="action-buttons">
        <button 
          onClick={handleRenderBackground}
          disabled={disabled}
          className="btn-primary"
        >
          Build Background PDF
        </button>
        
        <button 
          onClick={handleRenderLegacy}
          disabled={disabled}
          className="btn-secondary"
        >
          Render Legacy (V3/V5)
        </button>
      </div>
      
      {loading && <p className="status">Rendering...</p>}
    </div>
  );
}

export default RenderActions;

