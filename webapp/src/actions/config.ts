import type {Dispatch} from 'redux';

import {getCurrentUserId} from 'mattermost-redux/selectors/entities/users';

import type {GlobalState} from '@mattermost/types/store';

import ActionTypes from '@/action_types';
import PostType from '@/components/post_type';
import Client from '@/client';
import {postTypeComponent} from '@/selector';

import type {RegisterPostTypeComponentIdAction} from '@/types/actions';
import type {PluginRegistry, PluginStore} from '@/types/mattermost-webapp';
import type {PollConfiguration} from '@/types/poll';

export const configurationChange = (registry: PluginRegistry, store: PluginStore, data: PollConfiguration) => async (dispatch: Dispatch) => {
    let registeredComponentId = postTypeComponent(store.getState())?.id ?? '';
    if (data.experimentalui) {
        registeredComponentId = registry.registerPostTypeComponent('custom_matterpoll', PostType);
    } else {
        registry.unregisterPostTypeComponent(registeredComponentId);
        registeredComponentId = '';
    }

    const action: RegisterPostTypeComponentIdAction = {
        type: ActionTypes.REGISTER_POST_TYPE_COMPONENT_ID,
        data: {postTypeComponentId: registeredComponentId},
    };

    return dispatch(action);
};

export const fetchPluginConfiguration = (state: GlobalState) => {
    return async (): Promise<PollConfiguration | null> => {
        const currentUserId = getCurrentUserId(state);
        if (currentUserId) {
            return Client.getPluginConfiguration();
        }
        return null;
    };
};
