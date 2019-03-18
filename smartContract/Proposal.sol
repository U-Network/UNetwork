pragma solidity ^0.5.0;

/**
 * @title Math
 * @dev Assorted math operations
 */
library Math {
    /**
    * @dev Returns the largest of two numbers.
    */
    function max(uint256 a, uint256 b) internal pure returns (uint256) {
        return a >= b ? a : b;
    }

    /**
    * @dev Returns the smallest of two numbers.
    */
    function min(uint256 a, uint256 b) internal pure returns (uint256) {
        return a < b ? a : b;
    }

    /**
    * @dev Calculates the average of two numbers. Since these are integers,
    * averages of an even and odd number cannot be represented, and will be
    * rounded down.
    */
    function average(uint256 a, uint256 b) internal pure returns (uint256) {
        // (a + b) / 2 can overflow, so we distribute
        return (a / 2) + (b / 2) + ((a % 2 + b % 2) / 2);
    }
}

/**
 * @title SafeMath
 * @dev Math operations with safety checks that revert on error
 */
library SafeMath {
    int256 constant private INT256_MIN = -2**255;

    /**
    * @dev Multiplies two unsigned integers, reverts on overflow.
    */
    function mul(uint256 a, uint256 b) internal pure returns (uint256) {
        // Gas optimization: this is cheaper than requiring 'a' not being zero, but the
        // benefit is lost if 'b' is also tested.
        // See: https://github.com/OpenZeppelin/openzeppelin-solidity/pull/522
        if (a == 0) {
            return 0;
        }

        uint256 c = a * b;
        require(c / a == b);

        return c;
    }

    /**
    * @dev Multiplies two signed integers, reverts on overflow.
    */
    function mul(int256 a, int256 b) internal pure returns (int256) {
        // Gas optimization: this is cheaper than requiring 'a' not being zero, but the
        // benefit is lost if 'b' is also tested.
        // See: https://github.com/OpenZeppelin/openzeppelin-solidity/pull/522
        if (a == 0) {
            return 0;
        }

        require(!(a == -1 && b == INT256_MIN)); // This is the only case of overflow not detected by the check below

        int256 c = a * b;
        require(c / a == b);

        return c;
    }

    /**
    * @dev Integer division of two unsigned integers truncating the quotient, reverts on division by zero.
    */
    function div(uint256 a, uint256 b) internal pure returns (uint256) {
        // Solidity only automatically asserts when dividing by 0
        require(b > 0);
        uint256 c = a / b;
        // assert(a == b * c + a % b); // There is no case in which this doesn't hold

        return c;
    }

    /**
    * @dev Integer division of two signed integers truncating the quotient, reverts on division by zero.
    */
    function div(int256 a, int256 b) internal pure returns (int256) {
        require(b != 0); // Solidity only automatically asserts when dividing by 0
        require(!(b == -1 && a == INT256_MIN)); // This is the only case of overflow

        int256 c = a / b;

        return c;
    }

    /**
    * @dev Subtracts two unsigned integers, reverts on overflow (i.e. if subtrahend is greater than minuend).
    */
    function sub(uint256 a, uint256 b) internal pure returns (uint256) {
        require(b <= a);
        uint256 c = a - b;

        return c;
    }

    /**
    * @dev Subtracts two signed integers, reverts on overflow.
    */
    function sub(int256 a, int256 b) internal pure returns (int256) {
        int256 c = a - b;
        require((b >= 0 && c <= a) || (b < 0 && c > a));

        return c;
    }

    /**
    * @dev Adds two unsigned integers, reverts on overflow.
    */
    function add(uint256 a, uint256 b) internal pure returns (uint256) {
        uint256 c = a + b;
        require(c >= a);

        return c;
    }

    /**
    * @dev Adds two signed integers, reverts on overflow.
    */
    function add(int256 a, int256 b) internal pure returns (int256) {
        int256 c = a + b;
        require((b >= 0 && c >= a) || (b < 0 && c < a));

        return c;
    }

    /**
    * @dev Divides two unsigned integers and returns the remainder (unsigned integer modulo),
    * reverts when dividing by zero.
    */
    function mod(uint256 a, uint256 b) internal pure returns (uint256) {
        require(b != 0);
        return a % b;
    }
}

/**
 * @title Ownable
 * @dev The Ownable contract has an owner address, and provides basic authorization control
 * functions, this simplifies the implementation of "user permissions".
 */
contract Ownable {
    address private _owner;

    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    /**
     * @dev The Ownable constructor sets the original `owner` of the contract to the sender
     * account.
     */
    constructor () internal {
        _owner = msg.sender;
        emit OwnershipTransferred(address(0), _owner);
    }

    /**
     * @return the address of the owner.
     */
    function owner() public view returns (address) {
        return _owner;
    }

    /**
     * @dev Throws if called by any account other than the owner.
     */
    modifier onlyOwner() {
        require(isOwner());
        _;
    }

    /**
     * @return true if `msg.sender` is the owner of the contract.
     */
    function isOwner() public view returns (bool) {
        return msg.sender == _owner;
    }

    /**
     * @dev Allows the current owner to relinquish control of the contract.
     * @notice Renouncing to ownership will leave the contract without an owner.
     * It will not be possible to call the functions with the `onlyOwner`
     * modifier anymore.
     */
    function renounceOwnership() public onlyOwner {
        emit OwnershipTransferred(_owner, address(0));
        _owner = address(0);
    }

    /**
     * @dev Allows the current owner to transfer control of the contract to a newOwner.
     * @param newOwner The address to transfer ownership to.
     */
    function transferOwnership(address newOwner) public onlyOwner {
        _transferOwnership(newOwner);
    }

    /**
     * @dev Transfers control of the contract to a newOwner.
     * @param newOwner The address to transfer ownership to.
     */
    function _transferOwnership(address newOwner) internal {
        require(newOwner != address(0));
        emit OwnershipTransferred(_owner, newOwner);
        _owner = newOwner;
    }
}

contract Escrow is Ownable {
    using SafeMath for uint256;
    using Math for uint256;

    event Deposited(address indexed payee, uint256 weiAmount);
    event Withdrawn(address indexed payee, uint256 weiAmount);
    
    struct Assets {
        uint256 funds;
        uint256 startTime;
    }

    mapping(address => Assets) private _deposits;
    
    uint256 private _mortageDuration;
    
    constructor () internal {
        _mortageDuration = 1 days;
    }

    function depositsOf(address payee) public view returns (uint256) {
        return _deposits[payee].funds;
    }

    function _canDeposit() private pure {
    }

    function _canWithdraw(uint256 _payment, uint256 _withdrawTime) private view{
        require(_withdrawTime <= block.timestamp, "Time is not up yet.");
        require(_payment <= depositsOf(msg.sender), "You don't have that much money to withdraw.");
    }

    function _changeInfo(uint256 _time) internal {
        uint256 _startTime = _time.sub(_mortageDuration);
        _deposits[msg.sender].startTime = (_startTime.max(_deposits[msg.sender].startTime));
    }

    /**
    * @dev Stores the sent amount as credit to be withdrawn.
    */
    function deposit() public payable {
        uint256 _amount  = msg.value;
        address _payee   = msg.sender;
//        _canDeposit();
        _deposits[_payee].funds = _deposits[_payee].funds.add(_amount);
        _deposits[_payee].startTime = block.timestamp;
        emit Deposited(_payee, _amount);
    }

    /**
    * @dev Withdraw accumulated balance for a payee.
    */
    function withdraw(uint256 _payment) public {
        address payable _payee   = msg.sender;
        uint256 _withdrawTime = _deposits[_payee].startTime.add(_mortageDuration);
        _canWithdraw(_payment, _withdrawTime);
        
        _deposits[_payee].funds = _deposits[_payee].funds.sub(_payment);
        _payee.transfer(_payment);  
        emit Withdrawn(_payee, _payment);
    }
    
    function mortageInfo(address _payee) public view returns(uint256, uint256){
        return (_deposits[_payee].funds, _deposits[_payee].startTime);
    }
    
    function changeMortage(uint256 _duration) public onlyOwner {
        _mortageDuration = _duration;
    }
}

contract Proposal is Escrow {

    event Proposed(address _proposer, bytes _hash, uint256 _startTime);
    event Voted(address _voter, bytes _hash, uint8 _vote);
    
    struct ProposalInfo {
        address user;
        uint256 startTime;
        uint256 duration;
    }
    
    uint256 private _digit = 10 ** 18;
    // wei
    uint256 private _proposalCost;
    // wei
    uint256 private _voteCost;
    // 1 UUU : 1 vote
    uint256 private _voteProportion;
    
    // votes[_hashId][0] : result of downvote;
    // votes[_hahsId][1] : result of upvote; 
    mapping(uint256 => mapping(uint8 => uint256)) votes;
    mapping(uint256 => mapping(address => bool)) hasVoted;
    mapping(uint256 => ProposalInfo) proposals;
    
    constructor (uint256 _proposalcostI, uint256 _voteCostI) public payable{
        _proposalCost   = _proposalcostI;
        _voteCost       = _voteCostI;
        _voteProportion = 1;
    }
    
    modifier canPropose(bytes memory _hash, uint256 _startTime, uint256 _duration) {
        require(proposals[uint256(keccak256(_hash))].user == address(0), "This proposal has already been propesed.");
        require(msg.value >= _proposalCost, "You don't have enough coins to propose.");
        require(_duration <= 30 days);
        _;
    }
    
    modifier canVote(bytes memory _hash) {
        require(!hasVoted[uint256(keccak256(_hash))][msg.sender], "You have already voted.");
        require(depositsOf(msg.sender) >= _voteCost, "You don't have enough coins.");
        require(proposals[uint256(keccak256(_hash))].user != address(0), "This proposal hasn't been proposed.");
        require(block.timestamp <= proposals[uint256(keccak256(_hash))].startTime.add(proposals[uint256(keccak256(_hash))].duration), "This proposal has been closed.");
        require(block.timestamp >= proposals[uint256(keccak256(_hash))].startTime);
        _;
    }
    
    function propose(bytes memory _hash, uint256 _startTime, uint256 _duration) public payable 
    canPropose(_hash, _startTime, _duration){
        uint256 _hashId = uint256(keccak256(_hash));
        proposals[_hashId] = ProposalInfo(msg.sender, _startTime.max(block.timestamp), _duration);
        emit Proposed(msg.sender, _hash, _startTime);
    }
/*

    function upvote(bytes memory _hash) public canVote(_hash) {
        uint256 _hashId = uint256(keccak256(_hash));
        votes[_hashId][0] = votes[_hashId][0].add(depositsOf(msg.sender).div(_digit) * _voteProportion); 
        hasVoted[_hashId][msg.sender] = true;
    }
    
    function downvote(bytes memory _hash) public canVote(_hash) {
        uint256 _hashId = uint256(keccak256(_hash));
        votes[_hashId][1] = votes[_hashId][1].add(depositsOf(msg.sender).div(_digit) * _voteProportion); 
        hasVoted[_hashId][msg.sender] = true;
    }
 */   
 
    function ifVoted(bytes memory _hash) view public returns(bool){
        return hasVoted[uint256(keccak256(_hash))][msg.sender];
    }

    function vote(bytes memory _hash, uint8 _vote) public canVote(_hash) {
        uint256 _hashId = uint256(keccak256(_hash));
        votes[_hashId][_vote] = votes[_hashId][_vote].add(depositsOf(msg.sender).div(_digit) * _voteProportion); 
        hasVoted[_hashId][msg.sender] = true;
        _changeInfo(proposals[_hashId].startTime.add(proposals[_hashId].duration));
        emit Voted(msg.sender, _hash, _vote);
    }
 
    function proposalCost() public view returns(uint256) {
        return _proposalCost;
    }
    
    function voteCost() public view returns(uint256) {
        return _voteCost;
    }
 
    function results(bytes memory _hash, uint8 _vote) public view returns(uint256){
        return votes[uint256(keccak256(_hash))][_vote];
    }
    
    function changeProposalCost(uint256 _cost) public onlyOwner {
        _proposalCost = _cost;
    }
    
    function changeVoteCost(uint256 _cost) public onlyOwner {
        _voteCost = _cost;
    }
    
    function () external payable{
        deposit();
    }
}