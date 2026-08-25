import type {Dispatch} from 'redux';

import ActionTypes from '@/action_types';
import {fetchPollMetadata} from '@/actions/poll_metadata';

describe('test', () => {
    const mockSuccessResponse = {};
    let dispatch: jest.MockedFunction<Dispatch>;

    beforeEach(() => {
        const mockJsonPromise = Promise.resolve(mockSuccessResponse);
        const mockFetchPromise = Promise.resolve({
            json: () => mockJsonPromise,
        });
        global.fetch = jest.fn().mockImplementation(() => mockFetchPromise);

        dispatch = jest.fn();
    });

    it('success', async () => {
        const siteUrl = 'https://example.com:8065';
        const pollId = 'poll_id1';
        const expected = {
            type: ActionTypes.FETCH_POLL_METADATA,
            data: mockSuccessResponse,
        };

        await fetchPollMetadata(siteUrl, pollId)(dispatch);

        expect(dispatch).toHaveBeenCalledWith(expected);
    });

    it('fail, pollId is undefined', async () => {
        const siteUrl = 'https://example.com:8065';
        const pollId = undefined; // eslint-disable-line no-undefined

        await fetchPollMetadata(siteUrl, pollId)(dispatch);

        expect(dispatch).not.toHaveBeenCalled();
    });

    it('fail, pollId is empty', async () => {
        const siteUrl = 'https://example.com:8065';
        const pollId = '';

        await fetchPollMetadata(siteUrl, pollId)(dispatch);

        expect(dispatch).not.toHaveBeenCalled();
    });
});
