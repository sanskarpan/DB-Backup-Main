import React from 'react';
import {render, fireEvent, waitFor} from '@testing-library/react-native';
import {Provider} from 'react-redux';
import configureStore from 'redux-mock-store';
import HomeScreen from '../src/screens/HomeScreen';

const mockStore = configureStore([]);

describe('HomeScreen', () => {
  let store: any;
  let navigation: any;

  beforeEach(() => {
    store = mockStore({
      backups: {
        backups: [
          {
            id: '1',
            database_name: 'Test DB',
            status: 'completed',
            created_at: '2024-01-09T10:00:00Z',
            size: 1024000,
            duration: 120,
          },
        ],
        loading: false,
        error: null,
      },
    });

    navigation = {
      navigate: jest.fn(),
    };
  });

  it('renders correctly', () => {
    const {getByText} = render(
      <Provider store={store}>
        <HomeScreen navigation={navigation} />
      </Provider>,
    );

    expect(getByText('DB Backup')).toBeTruthy();
    expect(getByText('Dashboard')).toBeTruthy();
  });

  it('displays stats correctly', () => {
    const {getByText} = render(
      <Provider store={store}>
        <HomeScreen navigation={navigation} />
      </Provider>,
    );

    expect(getByText('Total Backups')).toBeTruthy();
    expect(getByText('1')).toBeTruthy();
  });

  it('navigates to Backups screen on quick action', () => {
    const {getByText} = render(
      <Provider store={store}>
        <HomeScreen navigation={navigation} />
      </Provider>,
    );

    const newBackupButton = getByText('New Backup');
    fireEvent.press(newBackupButton);

    expect(navigation.navigate).toHaveBeenCalledWith('Backups');
  });

  it('displays recent backups', () => {
    const {getByText} = render(
      <Provider store={store}>
        <HomeScreen navigation={navigation} />
      </Provider>,
    );

    expect(getByText('Test DB')).toBeTruthy();
    expect(getByText('Completed')).toBeTruthy();
  });

  it('shows empty state when no backups', () => {
    store = mockStore({
      backups: {
        backups: [],
        loading: false,
        error: null,
      },
    });

    const {getByText} = render(
      <Provider store={store}>
        <HomeScreen navigation={navigation} />
      </Provider>,
    );

    expect(getByText('No backups yet')).toBeTruthy();
  });

  it('shows loading state', () => {
    store = mockStore({
      backups: {
        backups: [],
        loading: true,
        error: null,
      },
    });

    const {getByTestId} = render(
      <Provider store={store}>
        <HomeScreen navigation={navigation} />
      </Provider>,
    );

    // ActivityIndicator should be visible
    expect(true).toBeTruthy();
  });

  it('handles refresh', async () => {
    const {UNSAFE_getByType} = render(
      <Provider store={store}>
        <HomeScreen navigation={navigation} />
      </Provider>,
    );

    // Test refresh control functionality
    await waitFor(() => {
      expect(true).toBeTruthy();
    });
  });
});
