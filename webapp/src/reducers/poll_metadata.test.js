import ActionTypes from 'action_types';
import {pollMetadata} from 'reducers/poll_metadata';

const initialState = {
    poll_id1: {
        user_id: 'user_id1',
        poll_id: 'poll_id1',
        admin_permission: false,
        voted_answers: ['answer1'],
    },
};
const additionalState = {
    user_id: 'user_id1',
    poll_id: 'poll_id2',
    admin_permission: true,
    voted_answers: [],
};

describe('vote reducers', () => {
    test('no action', () => expect(pollMetadata(undefined, {})).toEqual({})); // eslint-disable-line no-undefined
    test('no action with initial state', () => {
        expect(
            pollMetadata(initialState, {})
        ).toEqual(initialState);
    });
    test('action to add new poll', () => {
        expect(
            pollMetadata(
                initialState,
                {
                    type: ActionTypes.FETCH_POLL_METADATA,
                    data: {
                        user_id: 'user_id1',
                        poll_id: 'poll_id2',
                        admin_permission: true,
                        voted_answers: [],
                    },
                },
            ),
        ).toEqual({
            ...initialState,
            poll_id2: additionalState,
        });
    });
    test('action to add new answer', () => {
        const expected = initialState;
        expected.poll_id1.voted_answers = ['answer1', 'answer2'];

        expect(
            pollMetadata(
                initialState,
                {
                    type: ActionTypes.FETCH_POLL_METADATA,
                    data: {
                        user_id: 'user_id1',
                        poll_id: 'poll_id1',
                        admin_permission: false,
                        voted_answers: ['answer1', 'answer2'],
                    },
                },
            ),
        ).toEqual(expected);
    });
    test('action to update poll without empty poll_id', () => {
        expect(
            pollMetadata(
                initialState,
                {
                    type: ActionTypes.FETCH_POLL_METADATA,
                    data: {
                        user_id: 'user_id1',
                        poll_id: '',
                        admin_permission: false,
                        voted_answers: ['answer1', 'answer2'],
                    },
                },
            ),
        ).toEqual(initialState);
    });
    test('action with undefined', () => {
        expect(
            pollMetadata(
                initialState,
                {
                    type: ActionTypes.FETCH_POLL_METADATA,
                    data: undefined, // eslint-disable-line no-undefined
                },
            ),
        ).toEqual(initialState);
    });
});
