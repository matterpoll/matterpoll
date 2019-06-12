import {doPostAction} from 'mattermost-redux/actions/posts';

import PostType from './components/post_type';
import Manifest from './manifest';
import Client from './client';
import { postTypeComponent } from './selector';

export const VOTE_ANSWER = Manifest.PluginId + '_vote_answer';
export const FETCH_VOTED_ANSWERS = Manifest.PluginId + '_fetch_voted_answers';
export const REGIST_POST_TYPE_COMPONENT_ID = Manifest.PluginId + '_regist_post_type_conponent_id';

export const voteAnswer = (postId, actionId) => async (dispatch) => {
    return dispatch(doPostAction(postId, actionId));
};

export const configurationChange = (registry, store, data) => async (dispatch) => {
    let registeredComponentId = postTypeComponent(store.getState()) ? postTypeComponent(store.getState()).id : '';
    if (data.experimentalui) {
        registeredComponentId = registry.registerPostTypeComponent('custom_matterpoll', PostType);
    } else {
        registry.unregisterPostTypeComponent(registeredComponentId);
        registeredComponentId = '';
    }

    return dispatch({
        type: REGIST_POST_TYPE_COMPONENT_ID,
        data: {postTypeComponentId: registeredComponentId},
    })
}

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

export const fetchPluginConfiguration = () => {
    return async () => {
        let data;
        try {
            data = await Client.getPluginConfiguration();
        } catch (error) {
            return error;
        }
        return data;
    }
}