import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {voteAnswer} from 'actions/vote';

import ActionButton from './action_button';

function mapStateToProps() {
    return {};
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            voteAnswer,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(ActionButton);
