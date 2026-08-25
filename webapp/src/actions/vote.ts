import type {ThunkDispatch} from 'redux-thunk';

import {doPostAction} from 'mattermost-redux/actions/posts';
import type {MMReduxAction} from 'mattermost-redux/action_types';

import type {GlobalState} from '@mattermost/types/store';

export const voteAnswer = (postId: string, actionId: string) => async (dispatch: ThunkDispatch<GlobalState, unknown, MMReduxAction>) => {
    return dispatch(doPostAction(postId, actionId));
};
