import { createSlice, createAsyncThunk } from '@reduxjs/toolkit';

export interface RenderJob {
  jobId: string;
  documentId: string;
  action: 'background' | 'legacy';
  outputPath: string;
  status: 'pending' | 'success' | 'error';
  error?: string;
  timestamp: number;
}

interface RenderState {
  jobs: RenderJob[];
  loading: boolean;
  error: string | null;
}

const initialState: RenderState = {
  jobs: [],
  loading: false,
  error: null,
};

// Async thunks
export const renderBackground = createAsyncThunk(
  'render/renderBackground',
  async (documentId: string) => {
    const response = await fetch('/api/render/background', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ document_id: documentId }),
    });
    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || 'Failed to render background');
    }
    const data = await response.json();
    return {
      documentId,
      action: 'background' as const,
      ...data,
    };
  }
);

export const renderLegacy = createAsyncThunk(
  'render/renderLegacy',
  async (documentId: string) => {
    const response = await fetch('/api/render/legacy', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ document_id: documentId }),
    });
    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || 'Failed to render legacy');
    }
    const data = await response.json();
    return {
      documentId,
      action: 'legacy' as const,
      ...data,
    };
  }
);

const renderSlice = createSlice({
  name: 'render',
  initialState,
  reducers: {
    clearJobs(state) {
      state.jobs = [];
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(renderBackground.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(renderBackground.fulfilled, (state, action) => {
        state.loading = false;
        state.jobs.push({
          jobId: action.payload.job_id,
          documentId: action.payload.documentId,
          action: action.payload.action,
          outputPath: action.payload.output_path,
          status: 'success',
          timestamp: Date.now(),
        });
      })
      .addCase(renderBackground.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message || 'Failed to render background';
      })
      .addCase(renderLegacy.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(renderLegacy.fulfilled, (state, action) => {
        state.loading = false;
        state.jobs.push({
          jobId: action.payload.job_id,
          documentId: action.payload.documentId,
          action: action.payload.action,
          outputPath: action.payload.output_path,
          status: 'success',
          timestamp: Date.now(),
        });
      })
      .addCase(renderLegacy.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message || 'Failed to render legacy';
      });
  },
});

export const { clearJobs } = renderSlice.actions;
export default renderSlice.reducer;

