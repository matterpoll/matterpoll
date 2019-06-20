import ActionTypes from 'action_types';
import {votedAnswers} from 'reducers/vote';

const initialState = {
    poll_id1: {
        user_id: 'user_id1',
        poll_id: 'poll_id1',
        voted_answers: ['answer1'],
    },
};
const additionalState = {
    user_id: 'user_id1',
    poll_id: 'poll_id2',
    voted_answers: [],
};

describe('vote reducers', () => {
    test('no action', () => expect(votedAnswers(undefined, {})).toEqual({})); // eslint-disable-line no-undefined
    test('no action with initial state', () => {
        expect(
            votedAnswers(initialState, {})
        ).toEqual(initialState);
    });
    test('action to add new poll', () => {
        expect(
            votedAnswers(
                initialState,
                {
                    type: ActionTypes.FETCH_VOTED_ANSWERS,
                    data: {
                        user_id: 'user_id1',
                        poll_id: 'poll_id2',
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
            votedAnswers(
                initialState,
                {
                    type: ActionTypes.FETCH_VOTED_ANSWERS,
                    data: {
                        user_id: 'user_id1',
                        poll_id: 'poll_id1',
                        voted_answers: ['answer1', 'answer2'],
                    },
                },
            ),
        ).toEqual(expected);
    });
    test('action to update poll without empty poll_id', () => {
        expect(
            votedAnswers(
                initialState,
                {
                    type: ActionTypes.FETCH_VOTED_ANSWERS,
                    data: {
                        user_id: 'user_id1',
                        poll_id: '',
                        voted_answers: ['answer1', 'answer2'],
                    },
                },
            ),
        ).toEqual(initialState);
    });
    test('action with undefined', () => {
        expect(
            votedAnswers(
                initialState,
                {
                    type: ActionTypes.FETCH_VOTED_ANSWERS,
                    data: undefined, // eslint-disable-line no-undefined
                },
            ),
        ).toEqual(initialState);
    });
});
