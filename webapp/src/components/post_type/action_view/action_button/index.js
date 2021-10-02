import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';
import {getTheme} from 'mattermost-redux/selectors/entities/preferences';

import {voteAnswer} from 'actions/vote';

import ActionButton from './action_button';

function mapStateToProps(state) {
    return {
        theme: getTheme(state),
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            voteAnswer,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(ActionButton);
