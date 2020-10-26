import configureStore from 'redux-mock-store';

import ActionTypes from 'action_types';

import {fetchPollMetadata} from './poll_metadata';

const promisifyMiddleware = () => (next) => (action) => {
    return new Promise((resolve) => resolve(next(action)));
};
const middlewares = [promisifyMiddleware];
const mockStore = configureStore(middlewares);

describe('test', () => {
    const mockSuccessResponse = {};
    let store;

    beforeEach(() => {
        const mockJsonPromise = Promise.resolve(mockSuccessResponse);
        const mockFetchPromise = Promise.resolve({
            json: () => Promise.resolve(mockJsonPromise),
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

        store.dispatch(fetchPollMetadata(siteUrl, pollId)).
            then(() => {
                const actions = store.getActions();
                expect(actions[0]).toEqual(expected);
            });
    });
    it('fail, pollId is undefined', () => {
        const siteUrl = 'https://example.com:8065';
        const pollId = undefined; // eslint-disable-line no-undefined

        store.dispatch(fetchPollMetadata(siteUrl, pollId)).
            then(() => {
                const actions = store.getActions();
                expect(actions.length).toEqual(0);
            });
    });
    it('fail, pollId is empty', () => {
        const siteUrl = 'https://example.com:8065';
        const pollId = '';

        store.dispatch(fetchPollMetadata(siteUrl, pollId)).
            then(() => {
                const actions = store.getActions();
                expect(actions.length).toEqual(0);
            });
    });
});
