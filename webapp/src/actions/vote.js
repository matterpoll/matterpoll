import {doPostAction} from 'mattermost-redux/actions/posts';

import {id as pluginId} from 'manifest';
import ActionTypes from 'action_types';

export const voteAnswer = (postId, actionId) => async (dispatch) => {
    return dispatch(doPostAction(postId, actionId));
};

export const websocketHasVoted = (data) => async (dispatch) => {
    return dispatch({
        type: ActionTypes.FETCH_VOTED_ANSWERS,
        data: {
            user_id: data.user_id,
            poll_id: data.poll_id,
            voted_answers: data.voted_answers,
        },
    });
};

export const fetchVotedAnswers = (siteUrl, pollId) => async (dispatch) => {
    if (!pollId) {
        return;
    }

    let url = siteUrl.replace(/\/?$/, '');
    url = `${url}/plugins/${pluginId}/api/v1/polls/${pollId}/voted`;

    fetch(url).then((r) => r.json()).then((r) => {
        dispatch({
            type: ActionTypes.FETCH_VOTED_ANSWERS,
            data: r,
        });
    });
};
