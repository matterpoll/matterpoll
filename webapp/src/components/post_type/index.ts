import {connect} from 'react-redux';

import type {GlobalState} from '@mattermost/types/store';

import PostType from '@/components/post_type/post_type';
import {postTypeComponent} from '@/selector';

function mapStateToProps(state: GlobalState) {
    return {
        postTypeComponentId: postTypeComponent(state)?.id ?? '',
    };
}

export default connect(mapStateToProps)(PostType);
