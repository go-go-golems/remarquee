import { configureStore } from '@reduxjs/toolkit';
import documentsReducer from './documentsSlice';
import renderReducer from './renderSlice';
import validationReducer from './validationSlice';

export const store = configureStore({
  reducer: {
    documents: documentsReducer,
    render: renderReducer,
    validation: validationReducer,
  },
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;

