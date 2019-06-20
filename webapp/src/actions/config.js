import {doPostAction} from 'mattermost-redux/actions/posts';

import ActionTypes from 'action_types';
import PostType from 'components/post_type';
import Client from 'client';
import {postTypeComponent} from 'selector';

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
        type: ActionTypes.REGIST_POST_TYPE_COMPONENT_ID,
        data: {postTypeComponentId: registeredComponentId},
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
    };
};