import { createSlice, createAsyncThunk, type PayloadAction } from '@reduxjs/toolkit';

// Types matching backend API responses
export interface PageRef {
  index: number;
  pageId: string;
  sourcePdfPage: number;
  template: string;
  deleted: boolean;
}

export interface TestDocument {
  id: string;
  name: string;
  path: string;
  description: string;
  schema: string;
  docType: string;
  expectedPages: number;
}

export interface InspectResult {
  uuid: string;
  schema: string;
  docType: string;
  pageCount: number;
  pages: PageRef[];
  hasPayloadPDF: boolean;
}

interface DocumentsState {
  testDocuments: TestDocument[];
  selectedDocumentId: string | null;
  inspectResult: InspectResult | null;
  loading: boolean;
  error: string | null;
}

const initialState: DocumentsState = {
  testDocuments: [],
  selectedDocumentId: null,
  inspectResult: null,
  loading: false,
  error: null,
};

// Async thunks
export const fetchTestDocuments = createAsyncThunk(
  'documents/fetchTestDocuments',
  async () => {
    const response = await fetch('/api/test-documents');
    if (!response.ok) {
      throw new Error('Failed to fetch test documents');
    }
    return (await response.json()) as TestDocument[];
  }
);

export const fetchInspect = createAsyncThunk(
  'documents/fetchInspect',
  async (documentId: string) => {
    const response = await fetch(`/api/document/${documentId}/inspect`);
    if (!response.ok) {
      throw new Error('Failed to inspect document');
    }
    return (await response.json()) as InspectResult;
  }
);

const documentsSlice = createSlice({
  name: 'documents',
  initialState,
  reducers: {
    selectDocument(state, action: PayloadAction<string>) {
      state.selectedDocumentId = action.payload;
    },
    clearInspectResult(state) {
      state.inspectResult = null;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchTestDocuments.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchTestDocuments.fulfilled, (state, action) => {
        state.loading = false;
        state.testDocuments = action.payload;
      })
      .addCase(fetchTestDocuments.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message || 'Failed to fetch test documents';
      })
      .addCase(fetchInspect.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchInspect.fulfilled, (state, action) => {
        state.loading = false;
        state.inspectResult = action.payload;
      })
      .addCase(fetchInspect.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message || 'Failed to inspect document';
      });
  },
});

export const { selectDocument, clearInspectResult } = documentsSlice.actions;
export default documentsSlice.reducer;

