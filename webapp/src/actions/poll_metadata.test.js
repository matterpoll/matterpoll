import configureStore from 'redux-mock-store';
import thunk from 'redux-thunk';

import ActionTypes from '@/action_types';

import {fetchPollMetadata} from '@/actions/poll_metadata';

const middlewares = [thunk];
const mockStore = configureStore(middlewares);

describe('test', () => {
    const mockSuccessResponse = {};
    let store;

    beforeEach(() => {
        const mockJsonPromise = Promise.resolve(mockSuccessResponse);
        const mockFetchPromise = Promise.resolve({
            json: () => mockJsonPromise,
        });
        global.fetch = jest.fn().mockImplementation(() => mockFetchPromise);

        store = mockStore({});
    });

    it('success', async () => {
        const siteUrl = 'https://example.com:8065';
        const pollId = 'poll_id1';
        const expected = {
            type: ActionTypes.FETCH_POLL_METADATA,
            data: mockSuccessResponse,
        };

        await store.dispatch(fetchPollMetadata(siteUrl, pollId));
        const actions = store.getActions();
        expect(actions[0]).toEqual(expected);
    });

    it('fail, pollId is undefined', async () => {
        const siteUrl = 'https://example.com:8065';
        const pollId = undefined; // eslint-disable-line no-undefined

        await store.dispatch(fetchPollMetadata(siteUrl, pollId));
        const actions = store.getActions();
        expect(actions.length).toEqual(0);
    });

    it('fail, pollId is empty', async () => {
        const siteUrl = 'https://example.com:8065';
        const pollId = '';

        await store.dispatch(fetchPollMetadata(siteUrl, pollId));
        const actions = store.getActions();
        expect(actions.length).toEqual(0);
    });
});
