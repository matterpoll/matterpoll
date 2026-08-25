import type {ThunkDispatch} from 'redux-thunk';

import type {GlobalState} from '@mattermost/types/store';

import type {MMReduxAction} from 'mattermost-redux/action_types';
import {doPostAction} from 'mattermost-redux/actions/posts';

export const voteAnswer = (postId: string, actionId: string) => async (dispatch: ThunkDispatch<GlobalState, unknown, MMReduxAction>) => {
    return dispatch(doPostAction(postId, actionId));
};
