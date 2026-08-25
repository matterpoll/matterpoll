import {combineReducers} from 'redux';

import {pollMetadata} from '@/reducers/poll_metadata';
import {postTypeComponent} from '@/reducers/post_type';

export default combineReducers({
    postTypeComponent,
    pollMetadata,
});
