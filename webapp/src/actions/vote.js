import {doPostAction} from 'mattermost-redux/actions/posts';

export const voteAnswer = (postId, actionId) => async (dispatch) => {
    return dispatch(doPostAction(postId, actionId));
};
