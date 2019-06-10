import {doPostAction} from 'mattermost-redux/actions/posts';

import Manifest from './manifest';
import Client from './client';

export const VOTE_ANSWER = Manifest.PluginId + '_vote_answer';
export const FETCH_VOTED_ANSWERS = Manifest.PluginId + '_fetch_voted_answers';

export const voteAnswer = (postId, actionId) => async (dispatch) => {
    return dispatch(doPostAction(postId, actionId));
};

export const websocketHasVoted = (data) => async (dispatch) => {
    return dispatch({
        type: FETCH_VOTED_ANSWERS,
        data: {
            user_id: data.user_id,
            poll_id: data.poll_id,
            voted_answers: data.voted_answers,
        },
    });
};

export const fetchVotedAnswers = (siteUrl, pollId) => async (dispatch) => {
    if (typeof pollId === 'undefined' || pollId === '') {
        return;
    }

    let url = siteUrl;
    if (!url.endsWith('/')) {
        url += '/';
    }
    url = `${url}/plugins/${Manifest.PluginId}/api/v1/polls/${pollId}/voted`

    fetch(url).then((r) => r.json()).then((r) => {
        dispatch({
            type: FETCH_VOTED_ANSWERS,
            data: r,
        });

    });
};

export const fetchPluginSettings = () => {
    return async () => {
        let data;
        try {
            data = await Client.getPluginSettings();
        } catch (error) {
            return error;
        }
        return data;
    }
}