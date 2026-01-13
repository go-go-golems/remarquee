import { createSlice, createAsyncThunk, type PayloadAction } from '@reduxjs/toolkit';

export interface ValidationReview {
  status: 'pass' | 'fail' | 'unknown';
  notes: string;
}

export interface ValidationSession {
  documentId: string;
  actions: string[];
  review: ValidationReview;
  timestamp: number;
}

interface ValidationState {
  currentReview: ValidationReview;
  history: ValidationSession[];
  loading: boolean;
  error: string | null;
}

const initialState: ValidationState = {
  currentReview: {
    status: 'unknown',
    notes: '',
  },
  history: [],
  loading: false,
  error: null,
};

// Async thunks
export const submitValidation = createAsyncThunk(
  'validation/submitValidation',
  async (session: ValidationSession) => {
    const response = await fetch('/api/validation', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(session),
    });
    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || 'Failed to submit validation');
    }
    return await response.json();
  }
);

const validationSlice = createSlice({
  name: 'validation',
  initialState,
  reducers: {
    updateReviewStatus(state, action: PayloadAction<'pass' | 'fail' | 'unknown'>) {
      state.currentReview.status = action.payload;
    },
    updateReviewNotes(state, action: PayloadAction<string>) {
      state.currentReview.notes = action.payload;
    },
    resetReview(state) {
      state.currentReview = {
        status: 'unknown',
        notes: '',
      };
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(submitValidation.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(submitValidation.fulfilled, (state, action) => {
        state.loading = false;
        // Add to history (the action.payload should contain the saved session)
        state.history.push({
          ...action.meta.arg,
          timestamp: Date.now(),
        });
        // Reset current review
        state.currentReview = {
          status: 'unknown',
          notes: '',
        };
      })
      .addCase(submitValidation.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message || 'Failed to submit validation';
      });
  },
});

export const { updateReviewStatus, updateReviewNotes, resetReview } = validationSlice.actions;
export default validationSlice.reducer;

