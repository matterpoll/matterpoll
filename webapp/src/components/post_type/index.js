import {connect} from 'react-redux';

import {postTypeComponent} from '@/selector';

import PostType from '@/components/post_type/post_type';

function mapStateToProps(state) {
    return {
        postTypeComponentId: postTypeComponent(state) ? postTypeComponent(state).id : '',
    };
}

export default connect(mapStateToProps)(PostType);
