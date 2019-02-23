pragma solidity ^0.5.0;

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

contract Tube is Ownable { 
    uint256 public digit = 10 ** 18;

    uint256 private _readyTime;
    uint256 private _readyCoin;
    uint256 private _updateCoin;
    uint256	private	_updateTime;
    uint256 private _coinNumber;
    
    mapping(address => uint256) private _userUsed;
    
    event Update(uint256 time, uint256 coins);
    event GetCoin(address user, uint256 coins);

    constructor (uint256 _time, uint256 _coin, uint256 _number) public payable{
        _readyTime  = block.timestamp;
        _readyCoin  = 0;
        _updateTime = _time;
        _updateCoin = _coin;
        _coinNumber = _number;
    }

    function _cMin(uint256 _a, uint256 _b) internal view returns (uint256) {
        if (_a > _b) return _b;
        return _a;
    }

    modifier update() {
        if (_readyTime + updateTime() <= block.timestamp) {
            _readyTime += ((block.timestamp - _readyTime) / updateTime() + 1) * updateTime();
            _readyCoin = _cMin(address(this).balance, _updateCoin);
        }
        emit Update(block.timestamp, _coinNumber);
        _;
    }
    modifier canGet(address _payee) {
        require(_userUsed[_payee] < _readyTime, "You have already obtained free coins.");
        require(_readyCoin > _coinNumber, "There's no enough coins this turn.");
        _;
    }

    function getCoin(address payable _payee) public update canGet(_payee) {
        _userUsed[_payee] = block.timestamp;
        _readyCoin -= _coinNumber;
        _payee.transfer(_coinNumber * digit);
        emit GetCoin(_payee, _coinNumber);
    } 

    function changeUpdateTime(uint256 _time) public onlyOwner {
        _updateTime = _time;
    }

    function changeUpdateCoin(uint256 _coin) public onlyOwner {
        _updateCoin = _coin;
    }

    function changeCoinNumber(uint256 _number) public onlyOwner {
        _coinNumber = _number;
    }

    function coinNumber() public view returns (uint256) {
        return _coinNumber;
    }

    function updateTime() public view returns (uint256) {
        return _updateTime;
    }

    function updateCoin() public view returns (uint256) {
        return _updateCoin;
    }

    function readyTime() public view returns (uint256) {
        return _readyTime;
    }

    function readyCoin() public view returns (uint256) {
        return _readyCoin;
    }
    
    function funds() public view returns (uint256) {
        return address(this).balance / digit;
    }

    function userUsed(address _user) public view returns(uint256){
        return _userUsed[_user];
    }

    function () external payable {
        require(msg.sender == owner(), "You're not owner of this contract.");
    }
}
