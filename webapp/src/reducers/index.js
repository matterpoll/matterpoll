import {combineReducers} from 'redux';

import {postTypeComponent} from '@/reducers/post_type';
import {pollMetadata} from '@/reducers/poll_metadata';

export default combineReducers({
    postTypeComponent,
    pollMetadata,
});
