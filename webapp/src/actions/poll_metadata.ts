import type {Dispatch} from 'redux';

import ActionTypes from '@/action_types';
import {id as pluginId} from '@/manifest';
import type {FetchPollMetadataAction} from '@/types/actions';
import type {PollMetadata} from '@/types/poll';

export const websocketHasVoted = (data: PollMetadata) => async (dispatch: Dispatch) => {
    const action: FetchPollMetadataAction = {
        type: ActionTypes.FETCH_POLL_METADATA,
        data: {
            voted_answers: data.voted_answers,
            user_id: data.user_id,
            poll_id: data.poll_id,
            can_manage_poll: data.can_manage_poll,
            setting_progress: data.setting_progress,
            setting_public_add_option: data.setting_public_add_option,
        },
    };

    return dispatch(action);
};

export const fetchPollMetadata = (siteUrl: string, pollId?: string) => async (dispatch: Dispatch) => {
    if (!pollId) {
        return;
    }

    let url = siteUrl.replace(/\/?$/, '');
    url = `${url}/plugins/${pluginId}/api/v1/polls/${pollId}/metadata`;

    try {
        const resp = await fetch(url);
        const action: FetchPollMetadataAction = {
            type: ActionTypes.FETCH_POLL_METADATA,
            data: await resp.json(),
        };
        dispatch(action);
    } catch (err) {
        //eslint-disable-next-line no-console
        console.log('failed to fetch metadata: ', err);
    }
};
