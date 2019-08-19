import {getCurrentUserId} from 'mattermost-redux/selectors/entities/users';

import ActionTypes from 'action_types';
import PostType from 'components/post_type';
import Client from 'client';
import {postTypeComponent} from 'selector';

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

export const fetchPluginConfiguration = (state) => {
    return async () => {
        const currentUserId = getCurrentUserId(state);
        if (currentUserId) {
            return Client.getPluginConfiguration();
        }
        return null;
    };
};
